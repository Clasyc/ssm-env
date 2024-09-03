package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/chzyer/readline"
	"github.com/manifoldco/promptui"
)

const (
	maxRetries = 5
	retryDelay = 200 * time.Millisecond
)

type SSMManager struct {
	svc         *ssm.SSM
	prefix      string
	latestParam *ssm.Parameter
	quiet       bool
}

func main() {
	prefixFlag := flag.String("prefix", "", "SSM parameter prefix")
	quietFlag := flag.Bool("quiet", true, "Run in quiet mode (minimal output)")
	flag.Parse()

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	manager := &SSMManager{
		svc:   ssm.New(sess),
		quiet: *quietFlag,
	}

	manager.prefix = manager.getPrefix(*prefixFlag)

	for {
		if err := manager.run(); err != nil {
			if err.Error() == "user quit" {
				return
			}
			log.Printf("Error: %v", err)
		}
	}
}

func (m *SSMManager) run() error {
	params, err := m.fetchParameters()
	if err != nil {
		return fmt.Errorf("error fetching parameters: %w", err)
	}

	action, err := m.promptForAction(params)
	if err != nil {
		return fmt.Errorf("error prompting for action: %w", err)
	}

	switch action {
	case "quit":
		if !m.quiet {
			fmt.Println("Exiting...")
		}
		return fmt.Errorf("user quit")
	case "create":
		return m.createNewParameter()
	default:
		return m.updateParameter(params, action)
	}
}

func (m *SSMManager) getPrefix(flagValue string) string {
	if flagValue != "" {
		return ensureTrailingSlash(flagValue)
	}

	prompt := promptui.Prompt{
		Label: "Enter SSM parameter prefix",
	}

	result, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed: %v", err)
	}

	return ensureTrailingSlash(result)
}

func ensureTrailingSlash(s string) string {
	if !strings.HasSuffix(s, "/") {
		return s + "/"
	}
	return s
}

func (m *SSMManager) fetchParameters() ([]*ssm.Parameter, error) {
	var parameters []*ssm.Parameter
	var nextToken *string

	for {
		input := &ssm.GetParametersByPathInput{
			Path:           aws.String(m.prefix),
			Recursive:      aws.Bool(false),
			WithDecryption: aws.Bool(true),
			NextToken:      nextToken,
		}

		result, err := m.svc.GetParametersByPath(input)
		if err != nil {
			return nil, err
		}

		parameters = append(parameters, result.Parameters...)

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	// Update parameters with the latest param if it exists
	if m.latestParam != nil {
		for i, param := range parameters {
			if *param.Name == *m.latestParam.Name {
				parameters[i] = m.latestParam
				break
			}
		}
	}

	sort.Slice(parameters, func(i, j int) bool {
		return *parameters[i].Name < *parameters[j].Name
	})

	return parameters, nil
}

func (m *SSMManager) promptForAction(params []*ssm.Parameter) (string, error) {
	var items []string
	if len(params) == 0 {
		items = []string{
			"Create new variable",
			"Quit",
		}
	} else {
		items = append([]string{"Create new variable"}, m.formatParameters(params)...)
		items = append(items, "Quit")
	}

	prompt := promptui.Select{
		Label: "↑/↓: navigate • enter: select • ctrl+c: quit",
		Items: items,
		Size:  20,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "\u25B6 {{ . | cyan }}",
			Inactive: "  {{ . }}",
			Selected: "\u25B6 {{ . | cyan }}",
		},
		HideHelp: true,
	}

	index, result, err := prompt.Run()
	if err != nil {
		return "", err
	}

	if result == "Quit" {
		return "quit", nil
	}

	if index == 0 {
		return "create", nil
	}

	return result, nil
}

func (m *SSMManager) formatParameters(params []*ssm.Parameter) []string {
	var formatted []string
	for _, param := range params {
		name := strings.TrimPrefix(*param.Name, m.prefix)
		formatted = append(formatted, fmt.Sprintf("%s = %s", name, *param.Value))
	}
	return formatted
}

func (m *SSMManager) createNewParameter() error {
	name, err := m.promptForInput("Enter new parameter name")
	if err != nil {
		return err
	}

	value, err := m.promptForInput("Enter parameter value")
	if err != nil {
		return err
	}

	paramType, err := m.promptForParamType()
	if err != nil {
		return err
	}

	fullName := m.prefix + name

	err = m.updateParameterWithRetry(fullName, value, paramType)
	if err != nil {
		return fmt.Errorf("failed to create parameter: %w", err)
	}

	if !m.quiet {
		fmt.Println("Parameter created successfully.")
	}
	return nil
}

func (m *SSMManager) updateParameter(params []*ssm.Parameter, selection string) error {
	paramName := strings.SplitN(selection, " = ", 2)[0]
	fullName := m.prefix + paramName

	var param *ssm.Parameter
	for _, p := range params {
		if *p.Name == fullName {
			param = p
			break
		}
	}

	if param == nil {
		return fmt.Errorf("parameter not found: %s", fullName)
	}

	if !m.quiet {
		fmt.Printf("Updating parameter: %s\n", fullName)
		fmt.Printf("Current value: %s\n", *param.Value)
	}

	newValue, err := m.promptForNewValue(*param.Value)
	if err != nil {
		return err
	}

	if newValue == *param.Value {
		if !m.quiet {
			fmt.Println("No changes made.")
		}
		return nil
	}

	err = m.updateParameterWithRetry(fullName, newValue, *param.Type)
	if err != nil {
		return fmt.Errorf("error updating parameter: %w", err)
	}

	if !m.quiet {
		fmt.Println("Parameter updated successfully.")
	}

	m.latestParam = &ssm.Parameter{
		Name:  aws.String(fullName),
		Value: aws.String(newValue),
		Type:  param.Type,
	}

	return nil
}

func (m *SSMManager) promptForInput(label string) (string, error) {
	prompt := promptui.Prompt{
		Label: label,
	}

	return prompt.Run()
}

func (m *SSMManager) promptForParamType() (string, error) {
	prompt := promptui.Select{
		Label: "Select parameter type",
		Items: []string{"String", "StringList", "SecureString"},
	}

	_, result, err := prompt.Run()
	return result, err
}

func (m *SSMManager) promptForNewValue(currentValue string) (string, error) {
	var prompt string
	if m.quiet {
		prompt = "New value: "
	} else {
		prompt = "Enter new value (or press Enter to cancel): "
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     "/tmp/readline.tmp",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return "", err
	}
	defer rl.Close()

	newValue, err := rl.ReadlineWithDefault(currentValue)
	if err != nil {
		if err == readline.ErrInterrupt {
			return "", fmt.Errorf("update cancelled")
		}
		return "", err
	}

	return strings.TrimSpace(newValue), nil
}

func (m *SSMManager) updateParameterWithRetry(name, value, paramType string) error {
	for i := 0; i < maxRetries; i++ {
		_, err := m.svc.PutParameter(&ssm.PutParameterInput{
			Name:      aws.String(name),
			Value:     aws.String(value),
			Type:      aws.String(paramType),
			Overwrite: aws.Bool(true),
		})
		if err == nil {
			return nil
		}
		time.Sleep(retryDelay)
	}
	return fmt.Errorf("failed to update parameter after %d attempts", maxRetries)
}

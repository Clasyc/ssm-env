package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/chzyer/readline"
	"github.com/manifoldco/promptui"
)

func main() {
	prefixFlag := flag.String("prefix", "", "SSM parameter prefix")
	quietFlag := flag.Bool("quiet", false, "Run in quiet mode (minimal output)")
	flag.Parse()

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := ssm.New(sess)

	var prefix string
	if *prefixFlag != "" {
		prefix = *prefixFlag
	} else {
		prompt := promptui.Prompt{
			Label: "Enter SSM parameter prefix",
		}

		var err error
		prefix, err = prompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
	}

	// Ensure prefix ends with '/'
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var latestParam *ssm.Parameter

	for {
		// Fetch parameters
		params, err := fetchParameters(svc, prefix)
		if err != nil {
			fmt.Printf("Error fetching parameters: %v\n", err)
			return
		}

		// Update parameters with the latest param if it exists
		if latestParam != nil {
			for i, param := range params {
				if *param.Name == *latestParam.Name {
					params[i] = latestParam
					break
				}
			}
		}

		// Sort parameters alphabetically
		sort.Slice(params, func(i, j int) bool {
			return *params[i].Name < *params[j].Name
		})

		items := append([]string{"Create new variable"}, formatParameters(params)...)

		// Display parameters
		prompt := promptui.Select{
			Label: "↑/↓: navigate • enter: select • ctrl+c: quit",
			Items: items,
			Size:  20,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ . }}",
				Active:   "▶ {{ . | underline }}",
				Inactive: "  {{ . | greyOrNormal }}",
				Selected: "▶ {{ . | underline }}",
			},
			HideHelp: true,
		}

		funcMap := promptui.FuncMap
		funcMap["greyOrNormal"] = func(s string) string {
			if s == "Create new variable" {
				return promptui.Styler(promptui.FGBold)(s)
			}

			return promptui.Styler(promptui.FGYellow)(s)
		}
		funcMap["underline"] = func(s string) string {
			return promptui.Styler(promptui.FGUnderline)(s)
		}

		prompt.Templates.FuncMap = funcMap

		index, result, err := prompt.Run()

		if err != nil {
			if err == promptui.ErrInterrupt {
				fmt.Println("\nQuitting...")
				return
			}
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		if index == 0 {
			// Create new variable
			err = createNewParameter(svc, prefix, *quietFlag)
			if err != nil {
				fmt.Printf("Error creating parameter: %v\n", err)
			}
			continue
		}

		// Extract parameter name from result
		paramName := strings.SplitN(result, " = ", 2)[0]
		paramName = prefix + paramName

		// Get new value
		currentValue := *params[index-1].Value
		if !*quietFlag {
			fmt.Printf("Current value: %s\n", currentValue)
		}

		rl, err := readline.NewEx(&readline.Config{
			Prompt:          "Enter new value (or press Enter to cancel): ",
			HistoryFile:     "/tmp/readline.tmp",
			InterruptPrompt: "^C",
			EOFPrompt:       "exit",
		})
		if err != nil {
			panic(err)
		}
		defer rl.Close()

		newValue, err := rl.ReadlineWithDefault(currentValue)
		if err != nil {
			if err == readline.ErrInterrupt {
				if !*quietFlag {
					fmt.Println("\nUpdate cancelled.")
				}
				continue
			}
			fmt.Printf("Input failed %v\n", err)
			return
		}

		newValue = strings.TrimSpace(newValue)
		if newValue == "" || newValue == currentValue {
			if !*quietFlag {
				fmt.Println("No changes made.")
			}
			continue
		}

		// Update parameter
		err = updateParameter(svc, paramName, newValue, *params[index-1].Type)
		if err != nil {
			fmt.Printf("Error updating parameter: %v\n", err)
		} else {
			if !*quietFlag {
				fmt.Println("Parameter updated successfully.")
			}
			// Update latestParam
			latestParam = &ssm.Parameter{
				Name:  aws.String(paramName),
				Value: aws.String(newValue),
				Type:  params[index-1].Type,
			}
		}
	}
}

func fetchParameters(svc *ssm.SSM, prefix string) ([]*ssm.Parameter, error) {
	var parameters []*ssm.Parameter
	var nextToken *string

	for {
		input := &ssm.GetParametersByPathInput{
			Path:           aws.String(prefix),
			Recursive:      aws.Bool(false),
			WithDecryption: aws.Bool(true),
			NextToken:      nextToken,
		}

		result, err := svc.GetParametersByPath(input)
		if err != nil {
			return nil, err
		}

		parameters = append(parameters, result.Parameters...)

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return parameters, nil
}

func formatParameters(params []*ssm.Parameter) []string {
	var formatted []string
	for _, param := range params {
		name := strings.Split(*param.Name, "/")
		formatted = append(formatted, fmt.Sprintf("%s = %s", name[len(name)-1], *param.Value))
	}
	return formatted
}

func updateParameter(svc *ssm.SSM, name, value, paramType string) error {
	_, err := svc.PutParameter(&ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		Type:      aws.String(paramType),
		Overwrite: aws.Bool(true),
	})
	return err
}

func createNewParameter(svc *ssm.SSM, prefix string, quiet bool) error {
	namePrompt := promptui.Prompt{
		Label: "Enter new parameter name",
	}

	name, err := namePrompt.Run()
	if err != nil {
		return fmt.Errorf("name prompt failed: %v", err)
	}

	valuePrompt := promptui.Prompt{
		Label: "Enter parameter value",
	}

	value, err := valuePrompt.Run()
	if err != nil {
		return fmt.Errorf("value prompt failed: %v", err)
	}

	typePrompt := promptui.Select{
		Label: "Select parameter type",
		Items: []string{"String", "StringList", "SecureString"},
	}

	_, paramType, err := typePrompt.Run()
	if err != nil {
		return fmt.Errorf("type prompt failed: %v", err)
	}

	fullName := prefix + name

	_, err = svc.PutParameter(&ssm.PutParameterInput{
		Name:      aws.String(fullName),
		Value:     aws.String(value),
		Type:      aws.String(paramType),
		Overwrite: aws.Bool(false),
	})

	if err != nil {
		return fmt.Errorf("failed to create parameter: %v", err)
	}

	if !quiet {
		fmt.Println("Parameter created successfully.")
	}
	return nil
}

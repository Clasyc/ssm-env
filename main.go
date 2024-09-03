package main

import (
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
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := ssm.New(sess)

	prompt := promptui.Prompt{
		Label: "Enter SSM parameter prefix",
	}

	prefix, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	// Ensure prefix ends with '/'
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for {
		// Fetch parameters
		params, err := fetchParameters(svc, prefix)
		if err != nil {
			fmt.Printf("Error fetching parameters: %v\n", err)
			return
		}

		// Sort parameters alphabetically
		sort.Slice(params, func(i, j int) bool {
			return *params[i].Name < *params[j].Name
		})

		// Display parameters
		prompt := promptui.Select{
			Label:        "Select parameter to edit (or Ctrl+C to quit)",
			Items:        formatParameters(params),
			Size:         20,
			HideSelected: true,
		}

		index, result, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				fmt.Println("\nQuitting...")
				return
			}
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		// Extract parameter name from result
		paramName := strings.SplitN(result, " = ", 2)[0]
		paramName = prefix + paramName

		// Get new value
		currentValue := *params[index].Value
		fmt.Printf("Current value: %s\n", currentValue)

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
				fmt.Println("\nUpdate cancelled.")
				continue
			}
			fmt.Printf("Input failed %v\n", err)
			return
		}

		newValue = strings.TrimSpace(newValue)
		if newValue == "" || newValue == currentValue {
			fmt.Println("No changes made.")
			continue
		}

		// Update parameter
		err = updateParameter(svc, paramName, newValue)
		if err != nil {
			fmt.Printf("Error updating parameter: %v\n", err)
		} else {
			fmt.Println("Parameter updated successfully.")
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

func updateParameter(svc *ssm.SSM, name, value string) error {
	_, err := svc.PutParameter(&ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		Type:      aws.String("SecureString"),
		Overwrite: aws.Bool(true),
	})
	return err
}

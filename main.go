package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/manifoldco/promptui"
)

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := ssm.New(sess)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter SSM parameter prefix: ")
	prefix, _ := reader.ReadString('\n')
	prefix = strings.TrimSpace(prefix)

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
		fmt.Printf("Current value: %s\n", *params[index].Value) // Dereference the pointer here
		fmt.Print("Enter new value (or press Enter to cancel): ")
		newValue, _ := reader.ReadString('\n')
		newValue = strings.TrimSpace(newValue)

		if newValue == "" {
			fmt.Println("Update cancelled.")
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

# SSM Manager

SSM Manager is a command-line tool for handling application environment variables stored in AWS Systems Manager 
Parameter Store. It provides a more convenient way to manage these variables compared to using the AWS UI for SSM. 
The tool allows developers to view all parameters at once and edit multiple SSM parameters, similar to working with
a .env file. This approach is useful when dealing with numerous environment variables across different environments,
as it eliminates the need to click through the AWS console for each parameter. Being able to see all variables in a 
list format makes it easier to understand the full configuration of an application at a glance.

## Installation

### macOS

You can install ssm-manager using curl:

```bash
curl -L https://github.com/Clasyc/ssm-env/releases/latest/download/ssm-manager-darwin-amd64 -o ssm-manager
chmod +x ssm-manager
sudo mv ssm-manager /usr/local/bin/
```

### Ubuntu Linux

For Ubuntu Linux, use these commands:

```bash
curl -L https://github.com/Clasyc/ssm-env/releases/latest/download/ssm-manager-linux-amd64 -o ssm-manager
chmod +x ssm-manager
sudo mv ssm-manager /usr/local/bin/
```

After installation, you can run the tool by typing `ssm-manager` in your terminal.

### Local build

You can also build the tool locally by cloning the repository and running the following commands:

```bash
make build
```
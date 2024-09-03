# ssm-env

**ssm-env** is a command-line tool for handling application environment variables stored in AWS Systems Manager 
Parameter Store. It provides a more convenient way to manage these variables compared to using the AWS UI for SSM. 
The tool allows developers to view all parameters at once and edit multiple SSM parameters, similar to working with
a .env file. This approach is useful when dealing with numerous environment variables across different environments,
as it eliminates the need to click through the AWS console for each parameter. Being able to see all variables in a 
list format makes it easier to understand the full configuration of an application at a glance.

![demo.gif](demo.gif)

## Installation

### macOS

You can install `ssm-env` using these commands:

```bash
curl -L https://github.com/Clasyc/ssm-env/releases/download/v0.4.0/ssm-env-v0.4.0-darwin-amd64.tar.gz -o ssm-env.tar.gz
tar -xzvf ssm-env.tar.gz
chmod +x ssm-env
sudo mv ssm-env /usr/local/bin/ssm-env
rm ssm-env.tar.gz
```

### Ubuntu Linux

For Ubuntu Linux, use these commands:

```bash
curl -L https://github.com/Clasyc/ssm-env/releases/download/v0.4.0/ssm-env-v0.4.0-linux-amd64.tar.gz -o ssm-env.tar.gz
tar -xzvf ssm-env.tar.gz
chmod +x ssm-env
sudo mv ssm-env /usr/local/bin/ssm-env
rm ssm-env.tar.gz
```

After installation, you can run the tool by typing `ssm-env` in your terminal.

## AWS Configuration

`ssm-env` uses the default AWS profile configuration on your system. Ensure that you have configured your AWS 
credentials properly. You can do this by setting the `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and `AWS_REGION` environment variables, or by using the AWS CLI command `aws configure` to set your credentials.

For more information on how to configure your AWS credentials, you can refer to the [AWS CLI User Guide](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html).

## Usage

`ssm-env` can be run with the following options:

```bash
ssm-env [options]
```

### Options

- `-prefix string`: Specify the SSM parameter prefix (e.g., `/app/test/`)
- `-debug`: Run in debug mode with additional output
- `-secure`: Use secure mode to hide sensitive values (SSM SecureString)

### Examples

1. Run `ssm-env` interactively:
   ```bash
   ssm-env
   ```

2. Run `ssm-env` with a specific prefix:
   ```bash
   ssm-env -prefix /app/test/
   ```
   
## Local build

You can also build the tool locally by cloning the repository and running the following commands:

```bash
make build
make run
```
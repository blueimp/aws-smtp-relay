# AWS SMTP Relay
> SMTP server to relay emails via AWS SES API using IAM roles.

## Contents
- [Background](#background)
- [Docker](#docker)
- [Installation](#installation)
- [Usage](#usage)
  * [Options](#options)
  * [Region](#region)
  * [Credentials](#credentials)
  * [Logging](#logging)
- [Development](#development)
  * [Requirements](#requirements)
  * [Build](#build)
  * [Test](#test)
  * [Install](#install)
  * [Uninstall](#uninstall)
  * [Clean](#clean)
- [Credits](#credits)
- [License](#license)

## Background
[AWS SES](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/Welcome.html)
provides both an
[API](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/send-email-api.html)
and an [SMTP interface](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/send-email-smtp.html)
to send emails.

The SMTP interface is useful for applications that must use SMTP to send emails,
but it requires providing a set of
[SES SMTP Credentials](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/smtp-credentials.html).

For security reasons, using
[IAM roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html)
is preferable, but only possible with the SES API and not the SMTP interface.

This is where this project comes into play, as it provides an SMTP interface
that relays emails via SES API using IAM roles.

## Docker
This repository provides a sample [Dockerfile](Dockerfile) to build and run the
project in a container environment.

A prebuilt Docker image is also available on
[Docker Hub](https://hub.docker.com/r/blueimp/aws-smtp-relay/):

```sh
docker pull blueimp/aws-smtp-relay
```

## Installation
The `aws-smtp-relay` binary can be installed from source via
[go get](https://golang.org/cmd/go/):

```sh
go get github.com/blueimp/aws-smtp-relay
```

## Usage
By default, `aws-smtp-relay` listens on port `1025` on all interfaces when
started without arguments:

```sh
aws-smtp-relay
```

### Options
Available options can be listed the following way:

```sh
aws-smtp-relay --help
```

```
Usage of aws-smtp-relay:
  -a string
    	TCP listen address (default ":1025")
  -h string
    	Server hostname
  -n string
    	SMTP service name (default "AWS SMTP Relay")
```

### Region
The `AWS_REGION` must be set to configure the AWS SDK, e.g. by executing the
following command before starting `aws-smtp-relay`:

```sh
export AWS_REGION=eu-west-1
```

### Credentials
On [EC2](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/concepts.html) or
[ECS](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/Welcome.html),
security credentials for the IAM role are automatically retrieved:

* [IAM Roles for Amazon EC2](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html)
* [IAM Roles for Tasks](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-iam-roles.html)

### Logging
Requests are logged in `JSON` format to `stdout` with an empty `Error` property:

```json
{
  "Time": "2018-04-18T15:08:42.4388893Z",
  "IP": "172.17.0.1",
  "From": "alice@example.org",
  "To": [
    "bob@example.org"
  ],
  "Error": ""
}
```

Errors are logged in the same format to `stderr`, with the `Error` property set:

```json
{
  "Time": "2018-04-18T15:08:42.4388893Z",
  "IP": "172.17.0.1",
  "From": "alice@example.org",
  "To": [
    "bob@example.org"
  ],
  "Error": "MissingRegion: could not find region configuration"
}
```

## Development

### Requirements
First, clone the project via `go get` and then switch into its source directory:

```sh
go get github.com/blueimp/aws-smtp-relay
cd "$GOPATH/src/github.com/blueimp/aws-smtp-relay"
```

*Please note:*  
This project relies on [vgo](https://github.com/golang/go/wiki/vgo) for
automatic dependency resolution.

To use the original go tool instead, export the following environment variable:

```sh
export GO_CLI=go
```

And install the project dependencies:

```sh
go get ./...
```

### Build
To build the project, run
[Make](https://en.wikipedia.org/wiki/Make_\(software\)) in the repository
directory, which creates the `aws-smtp-relay` binary:

```sh
make
```

### Test
All components come with unit tests, which can be executed the following way:

```sh
make test
```

Sending mails can also be tested with the provided [mail shell script](mail.sh):

```sh
echo TEXT | ./mail.sh -p 1025 -f alice@example.org -t bob@example.org
```

See also
[Testing Amazon SES Email Sending](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/mailbox-simulator.html).

### Install
The binary can also be built and installed in `$GOPATH/bin/` with the following
command:

```sh
make install
```

### Uninstall
The uninstall command removes the binary from `$GOPATH/bin/`:

```sh
make uninstall
```

### Clean
To remove any build artifacts, run the following:

```sh
make clean
```

## Credits
Includes the [smtpd](https://github.com/mhale/smtpd) package by Mark Hale.

## License
Released under the [MIT license](LICENSE.txt).

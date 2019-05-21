# AWS SMTP Relay
> SMTP server to relay emails via AWS SES or Pinpoint API using IAM roles.

## Contents
- [Background](#background)
- [Docker](#docker)
- [Installation](#installation)
- [Usage](#usage)
  * [Options](#options)
  * [Authentication](#authentication)
  * [TLS](#tls)
  * [Region](#region)
  * [Credentials](#credentials)
  * [Logging](#logging)
- [Development](#development)
  * [Build](#build)
  * [Test](#test)
  * [Install](#install)
  * [Uninstall](#uninstall)
  * [Clean](#clean)
- [Credits](#credits)
- [License](#license)

## Background
[AWS SES](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/Welcome.html)
and [Pinpoint](https://docs.aws.amazon.com/sdk-for-go/api/service/pinpointemail/) provide an
[API](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/send-email-api.html)
and an [SMTP interface](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/send-email-smtp.html)
to send emails.

The SMTP interface is useful for applications that must use SMTP to send emails,
but it requires providing a set of
[SES SMTP Credentials](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/smtp-credentials.html), 
and these. credentials are shared to the Pinpoint API.

For security reasons, using
[IAM roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html)
is preferable, but only possible with the SES and Pinpoint API's and not the SMTP interface.

This is where this project comes into play, as it provides an SMTP interface
that relays emails via SES or Pinpoint API using IAM roles.

## Docker
This repository provides a sample [Dockerfile](Dockerfile) to build and run the
project in a container environment.

A prebuilt Docker image is also available on
[Docker Hub](https://hub.docker.com/r/blueimp/aws-smtp-relay/):

```sh
docker run blueimp/aws-smtp-relay --help
```

## Installation
The `aws-smtp-relay` binary can be installed from source via
[go get](https://golang.org/cmd/go/):

```sh
go get github.com/blueimp/aws-smtp-relay
```

## Usage
By default, `aws-smtp-relay` listens on port `1025` on all interfaces as open
relay (without authentication) when started without arguments:

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
  -c string
    	TLS cert file
  -h string
    	Server hostname
  -i string
    	Allowed client IPs (comma-separated)
  -k string
    	TLS key file
  -n string
  -p    Use AWS Pinpoint instead of SES
    	SMTP service name (default "AWS SMTP Relay")
  -s	Require TLS via STARTTLS extension
  -t	Listen for incoming TLS connections only
  -u string
    	Authentication username
```

### Authentication
To require authentication, supply the `-u username` option along with a
[bcrypt](https://en.wikipedia.org/wiki/Bcrypt) encrypted password as
`BCRYPT_HASH` environment variable:

```sh
PASSWORD_HASH=$(htpasswd -bnBC 10 '' password | tr -d ':\n')

BCRYPT_HASH="$PASSWORD_HASH" aws-smtp-relay -u username
```

To limit the allowed IP addresses, supply a comma-separated list as `-i ips`
option:

```sh
aws-smtp-relay -i 127.0.0.1,::1
```

### TLS
Edit the [openssl config file](tls/openssl.conf) and change `localhost` to your
server hostname.

Generate a self-signed certificate with a passphrase encrypted key:

```sh
openssl req -new -x509 -config tls/openssl.conf -days 24855 \
  -out tls/default.crt \
  -keyout /dev/stdout |
  openssl rsa -aes256 -out tls/default.key
```

**Please note**:  
> Encrypted key files are only supported if they contain a `DEK-Info` header,
> stating the encryption method used.  
> The `openssl req` command does not create this header if encryption is
> enabled, which is why we pipe the unencrypted key output to the
> `openssl rsa` command, which outputs an encrypted key file with the required
> `DEK-Info` header.

The key file passphrase must be provided as `TLS_KEY_PASS` environment variable:

```sh
TLS_KEY_PASS="$PASSPHRASE" aws-smtp-relay -c tls/default.crt -k tls/default.key
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

### Build
First, clone the project and then switch into its source directory:

```sh
git clone https://github.com/blueimp/aws-smtp-relay.git
cd aws-smtp-relay
```

*Please note:*  
This project relies on [Go modules](https://github.com/golang/go/wiki/Modules)
for automatic dependency resolution.

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

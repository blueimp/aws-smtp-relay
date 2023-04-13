# AWS SMTP Relay

> SMTP server to relay emails via Amazon SES or Amazon Pinpoint using IAM roles.

**Contents**

- [Background](#background)
- [Docker](#docker)
- [Installation](#installation)
- [Usage](#usage)
  - [Options](#options)
  - [Authentication](#authentication)
    - [User](#user)
    - [IP](#ip)
  - [TLS](#tls)
  - [Filtering](#filtering)
    - [Senders](#senders)
    - [Recipients](#recipients)
  - [Region](#region)
  - [Credentials](#credentials)
  - [Logging](#logging)
- [Development](#development)
  - [Build](#build)
  - [Lint](#lint)
  - [Test](#test)
  - [Install](#install)
  - [Uninstall](#uninstall)
  - [Clean](#clean)
- [Dependencies](#dependencies)
- [License](#license)

## Background

[Amazon SES](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/Welcome.html)
and
[Amazon Pinpoint](https://docs.aws.amazon.com/pinpoint/latest/developerguide/welcome.html)
both provide an API and an SMTP interface to send emails:

- [SES Email API](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/send-email-api.html)
- [SES SMTP interface](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/send-email-smtp.html)
- [Pinpoint EmailAPI](https://docs.aws.amazon.com/pinpoint/latest/developerguide/send-messages-email-sdk.html)
- [Pinpoint SMTP interface](https://docs.aws.amazon.com/pinpoint/latest/developerguide/send-messages-email-smtp.html)

The SMTP interface is useful for applications that must use SMTP to send emails,
but it requires providing a set of SMTP credentials:

- [SES SMTP Credentials](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/smtp-credentials.html)
- [Pinpoint SMTP Credentials](https://docs.aws.amazon.com/pinpoint/latest/userguide/channels-email-send-smtp.html#channels-email-send-smtp-credentials)

For security reasons, using
[IAM roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html) is
preferable, but only possible with the Email API and not the SMTP interface.

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
  -d string
        Denied recipient emails regular expression
  -e string
        Amazon SES Configuration Set Name
  -h string
        Server hostname
  -i string
        Allowed client IPs (comma-separated)
  -k string
        TLS key file
  -l string
        Allowed sender emails regular expression
  -n string
        SMTP service name (default "AWS SMTP Relay")
  -o string
    	  Role to Assume, If empty it uses local credentials
  -r string
        Relay API to use (ses|pinpoint) (default "ses")
  -s    Require TLS via STARTTLS extension
  -t    Listen for incoming TLS connections only
  -u string
        Authentication username
```

### Authentication

#### User

The supported user-based SMTP authentication mechanisms and their required
configuration settings (see also
[RFC 4954](https://tools.ietf.org/html/rfc4954#section-9)):

| Mechanism  | [TLS](#tls) | User | Hash | Pass |
| ---------- | ----------- | ---- | ---- | ---- |
| `LOGIN`    | Yes         | Yes  | Yes  | No   |
| `PLAIN`    | Yes         | Yes  | Yes  | No   |
| `CRAM-MD5` | No          | Yes  | No   | Yes  |

Authentication can be enabled for `LOGIN` and `PLAIN` mechanisms by configuring
[TLS](#tls) and a username and providing the
[bcrypt](https://en.wikipedia.org/wiki/Bcrypt) encrypted password as
`BCRYPT_HASH` environment variable:

```sh
export BCRYPT_HASH=$(htpasswd -bnBC 10 '' password | tr -d ':\n')
export TLS_KEY_PASS="$PASSPHRASE"

aws-smtp-relay -c tls/default.crt -k tls/default.key -u username
```

If the password is provided as plain text `PASSWORD` environment variable, it
will also enable the `CRAM-MD5` authentication mechanism:

```sh
export PASSWORD=password
export TLS_KEY_PASS="$PASSPHRASE"

aws-smtp-relay -c tls/default.crt -k tls/default.key -u username
```

Without [TLS](#tls) configuration, only `CRAM-MD5` will be enabled:

```sh
export PASSWORD=password

aws-smtp-relay -u username
```

**Please note**:

> It is not recommended to provide the password as plain text environment
> variable, nor to configure the SMTP server without [TLS](#tls) support.

#### IP

To limit the allowed IP addresses, supply a comma-separated list via `-i ips`
option:

```sh
aws-smtp-relay -i 127.0.0.1,::1
```

**Please note**:

> To authorize their IP, clients must use a supported SMTP authentication
> mechanism, e.g. `LOGIN` or `PLAIN` via [TLS](#tls) or `CRAM-MD5` on
> unencrypted connections.  
> This is required even if no user authentication is configured on the server,
> although in this case the credentials can be chosen freely by the client.

### TLS

Configure [TLS](https://en.wikipedia.org/wiki/Transport_Layer_Security) with the
following steps:

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
> enabled, which is why we pipe the unencrypted key output to the `openssl rsa`
> command, which outputs an encrypted key file with the required `DEK-Info`
> header.

Provide the key file passphrase as `TLS_KEY_PASS` environment variable and the
cert and key file as command-line arguments:

```sh
TLS_KEY_PASS="$PASSPHRASE" aws-smtp-relay -c tls/default.crt -k tls/default.key
```

**Please note**:

> It is recommended to require TLS via `STARTTLS` extension (`-s` option flag)
> or to configure the server to listen for incoming TLS connections only (`-t`
> option flag).

### Filtering

#### Senders

To limit the allowed sender email addresses, provide an allow list as
[regular expression](https://golang.org/pkg/regexp/syntax/) via `-l regexp`
option:

```sh
aws-smtp-relay -l '@example\.org$'
```

By default, all sender email addresses are allowed.

#### Recipients

To deny certain recipient email addresses, provide a deny list as
[regular expression](https://golang.org/pkg/regexp/syntax/) via `-d regexp`
option:

```sh
aws-smtp-relay -d 'admin@example\.org$'
```

By default, all recipient email addresses are allowed.

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

- [IAM Roles for Amazon EC2](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html)
- [IAM Roles for Tasks](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-iam-roles.html)

### Logging

Requests are logged in `JSON` format to `stdout` with the `Error` property set
to `null`:

```json
{
  "Time": "2018-04-18T15:08:42.4388893Z",
  "IP": "172.17.0.1",
  "From": "alice@example.org",
  "To": ["bob@example.org"],
  "Error": null
}
```

Errors are logged in the same format to `stderr`, with the `Error` property set
to a `string` value:

```json
{
  "Time": "2018-04-18T15:08:42.4388893Z",
  "IP": "172.17.0.1",
  "From": "alice@example.org",
  "To": ["bob@example.org"],
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

**Please note**:

> This project relies on [Go modules](https://github.com/golang/go/wiki/Modules)
> for automatic dependency resolution.

To build the project, run
[Make](<https://en.wikipedia.org/wiki/Make_(software)>) in the repository
directory, which creates the `aws-smtp-relay` binary:

```sh
make
```

### Lint

To lint the source code, first install [staticcheck](https://staticcheck.io/):

```sh
go install honnef.co/go/tools/cmd/staticcheck@latest
```

Then run the following command:

```sh
make lint
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

**Please note**:

> The provided shell script only supports the `LOGIN` authentication mechanism.

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

## Dependencies

- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto)
- [github.com/mhale/smtpd](https://github.com/mhale/smtpd)
- [github.com/aws/aws-sdk-go](https://github.com/aws/aws-sdk-go)

## License

Released under the [MIT license](LICENSE.txt).

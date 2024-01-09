module github.com/blueimp/aws-smtp-relay

go 1.21

require (
	github.com/aws/aws-sdk-go-v2/service/pinpointemail v1.12.4
	github.com/aws/aws-sdk-go-v2/service/ses v1.15.3
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.6.0
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.10 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.15 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.18.5 // indirect
)

require (
	github.com/aws/aws-sdk-go-v2 v1.17.5 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.18.15
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.30.5
	github.com/aws/aws-sdk-go-v2/service/sqs v1.20.4
	github.com/aws/smithy-go v1.13.5 // indirect
	github.com/emersion/go-smtp v0.16.0
	github.com/google/uuid v1.3.0
	github.com/jmespath/go-jmespath v0.4.0 // indirect
)

replace github.com/emersion/go-smtp v0.16.0 => github.com/mabels/go-smtp v0.0.0-20230331132253-992d2562fbaf

// replace github.com/emersion/go-smtp v0.16.0 => ../go-smtp

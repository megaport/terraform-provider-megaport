package provider

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	megaport "github.com/megaport/megaportgo"
)

// Set by Provider Config, is 10 minutes by default.
var waitForTime time.Duration

// megaportProviderModel maps provider schema data to a Go type.
type megaportProviderModel struct {
	Environment       types.String `tfsdk:"environment"`
	AccessKey         types.String `tfsdk:"access_key"`
	SecretKey         types.String `tfsdk:"secret_key"`
	TermsAccepted     types.Bool   `tfsdk:"accept_purchase_terms"`
	CancelAtEndOfTerm types.Bool   `tfsdk:"cancel_at_end_of_term"`
	WaitTime          types.Int64  `tfsdk:"wait_time"`
	AWSEnabled        types.Bool   `tfsdk:"aws_enabled"`
	AWSRegion         types.String `tfsdk:"aws_region"`
	AWSAccessKey      types.String `tfsdk:"aws_access_key"`
	AWSSecretKey      types.String `tfsdk:"aws_secret_key"`
	AWSSessionToken   types.String `tfsdk:"aws_session_token"`
	AWSProfile        types.String `tfsdk:"aws_profile"`
	AWSAssumeRoleARN  types.String `tfsdk:"aws_assume_role_arn"`
	AWSExternalID     types.String `tfsdk:"aws_external_id"`
	AWSConfiguration  types.Object `tfsdk:"aws_configuration"`
}

type awsConfigurationModel struct {
	AWSRegion        types.String `tfsdk:"aws_region"`
	AWSAccessKey     types.String `tfsdk:"aws_access_key"`
	AWSSecretKey     types.String `tfsdk:"aws_secret_key"`
	AWSSessionToken  types.String `tfsdk:"aws_session_token"`
	AWSProfile       types.String `tfsdk:"aws_profile"`
	AWSAssumeRoleARN types.String `tfsdk:"aws_assume_role_arn"`
	AWSExternalID    types.String `tfsdk:"aws_external_id"`
}

type awsConfig struct {
	Region        string
	AccessKey     string
	SecretKey     string
	SessionToken  string
	Enabled       bool
	Profile       string
	AssumeRoleARN string
	ExternalID    string
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &megaportProvider{}
)

type megaportProviderData struct {
	client            *megaport.Client
	awsConfig         *awsConfig
	cancelAtEndOfTerm bool
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &megaportProvider{
			version: version,
		}
	}
}

// megaportProvider is the provider implementation.
type megaportProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Metadata returns the provider type name.
func (p *megaportProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "megaport"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *megaportProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"environment": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("production", "staging", "development"),
				},
			},
			"access_key": schema.StringAttribute{
				Optional:    true,
				Description: "The API access key. Can also be set using the environment variable MEGAPORT_ACCESS_KEY",
			},
			"secret_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The API secret key. Can also be set using the environment variable MEGAPORT_SECRET_KEY",
			},
			"accept_purchase_terms": schema.BoolAttribute{
				Optional:    true,
				Description: "Indicates acceptance of the Megaport API terms, this is required to use the provider. Can also be set using the environment variable MEGAPORT_ACCEPT_PURCHASE_TERMS",
			},
			"aws_configuration": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Configuration block for AWS integration to manage AWS DirectConnect resources",
				Attributes: map[string]schema.Attribute{
					"aws_region": schema.StringAttribute{
						Optional:    true,
						Description: "AWS region to use for DirectConnect API calls",
					},
					"aws_access_key": schema.StringAttribute{
						Optional:    true,
						Description: "AWS access key for DirectConnect resource management",
					},
					"aws_secret_key": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "AWS secret key for DirectConnect resource management",
					},
					"aws_session_token": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "AWS session token for temporary credentials",
					},
					"aws_profile": schema.StringAttribute{
						Optional:    true,
						Description: "AWS profile to use from shared credentials file",
					},
					"aws_assume_role_arn": schema.StringAttribute{
						Optional:    true,
						Description: "ARN of role to assume for AWS operations",
					},
					"aws_external_id": schema.StringAttribute{
						Optional:    true,
						Description: "External ID to use when assuming a role",
					},
				},
			},
			"wait_time": schema.Int64Attribute{
				Description: "The time to wait in minutes for creating and updating resources in Megaport API. Default value is 10.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"cancel_at_end_of_term": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, resources will be marked for cancellation at the end of their billing term rather than immediately. Default is false (immediate cancellation). Please note that this is only applicable to resources that support cancellation at the end of the term, which is currently only the case for Single Ports and LAG Ports. For other resources, this attribute will be ignored.",
			},
		},
	}
}
func (p *megaportProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Megaport API client")

	// Retrieve provider data from configuration
	var config megaportProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Environment.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("environment"),
			"Unknown Megaport API environment",
			"The provider cannot create the Megaport API client as there is an unknown configuration value for the Megaport API environment. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the MEGAPORT_ENVIRONMENT environment variable.",
		)
	}

	if config.AccessKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("access_key"),
			"Unknown Megaport API access key",
			"The provider cannot create the Megaport API client as there is an unknown configuration value for the Megaport API access_key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the MEGAPORT_ACCESS_KEY environment variable.",
		)
	}

	if config.SecretKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret_key"),
			"Unknown Megaport API secret key",
			"The provider cannot create the Megaport API client as there is an unknown configuration value for the Megaport API secret key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the MEGAPORT_SECRET_KEY environment variable.",
		)
	}

	if config.TermsAccepted.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("accept_purchase_terms"),
			"Unknown purchase terms key",
			"The provider cannot create the Megaport API client as there is an unknown configuration value for the Megaport API purchase terms acceptance "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the MEGAPORT_ACCEPT_PURCHASE_TERMS environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	environment := os.Getenv("MEGAPORT_ENVIRONMENT")
	accessKey := os.Getenv("MEGAPORT_ACCESS_KEY")
	secretKey := os.Getenv("MEGAPORT_SECRET_KEY")
	acceptTerms := false
	waitTime := 10
	if strings.ToLower(os.Getenv("MEGAPORT_ACCEPT_PURCHASE_TERMS")) == "true" ||
		strings.ToLower(os.Getenv("MEGAPORT_ACCEPT_PURCHASE_TERMS")) == "yes" {
		acceptTerms = true
	}

	if !config.Environment.IsNull() {
		environment = config.Environment.ValueString()
	}

	if !config.AccessKey.IsNull() {
		accessKey = config.AccessKey.ValueString()
	}

	if !config.SecretKey.IsNull() {
		secretKey = config.SecretKey.ValueString()
	}

	if !config.TermsAccepted.IsNull() {
		acceptTerms = config.TermsAccepted.ValueBool()
	}

	if !config.WaitTime.IsNull() {
		waitTime = int(config.WaitTime.ValueInt64())
	}

	cancelAtEndOfTerm := false
	if !config.CancelAtEndOfTerm.IsNull() {
		cancelAtEndOfTerm = config.CancelAtEndOfTerm.ValueBool()
	}

	ctx = tflog.SetField(ctx, "environment", environment)
	ctx = tflog.SetField(ctx, "access_key", accessKey)
	ctx = tflog.SetField(ctx, "secret_key", secretKey)
	ctx = tflog.SetField(ctx, "terms_accepted", acceptTerms)
	ctx = tflog.SetField(ctx, "wait_time", waitTime)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "secret_key")

	tflog.Debug(ctx, "Creating Megaport client")

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	// Validate and set the correct environment
	var megaportGoEnv megaport.Environment
	switch environment {
	case "", "staging":
		megaportGoEnv = megaport.EnvironmentStaging
	case "production":
		megaportGoEnv = megaport.EnvironmentProduction
	case "development":
		megaportGoEnv = megaport.EnvironmentDevelopment
	default:
		resp.Diagnostics.AddAttributeError(
			path.Root("environment"),
			"Invalid Megaport environment",
			fmt.Sprintf("The provider cannot create the Megaport API client as there is an invalid value for the environment: \"%s\")", environment),
		)
	}

	if accessKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("access_key"),
			"Missing Megaport API access key",
			"The provider cannot create the Megaport API client as there is a missing or empty value for the Megaport API access key. "+
				"Set the access_key value in the configuration or use the MEGAPORT_ACCESS_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if secretKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret_key"),
			"Missing Megaport API secret key",
			"The provider cannot create the Megaport API client as there is a missing or empty value for the Megaport API secret key. "+
				"Set the secret_key value in the configuration or use the MEGAPORT_SECRET_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if !acceptTerms {
		resp.Diagnostics.AddAttributeError(
			path.Root("accept_purchase_terms"),
			"Missing Megaport API terms acceptance",
			"The provider cannot create the Megaport API client as there is a missing or empty value for the Megaport API terms acceptance. "+
				"Set the accept_purchase_terms value in the configuration or use the MEGAPORT_ACCEPT_PURCHASE_TERMS environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Build a useragent string with some useful information about the client
	userAgent := fmt.Sprintf("Terraform/%s terraform-provider-megaport/%s go/%s (%s %s)", req.TerraformVersion, p.version, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	waitForTime = (time.Duration(waitTime) * time.Minute)
	megaportClient, err := megaport.New(nil,
		megaport.WithEnvironment(megaportGoEnv),
		megaport.WithCredentials(accessKey, secretKey),
		megaport.WithLogHandler(tfhandler{}),
		megaport.WithCustomHeaders(map[string]string{
			"x-app": "terraform",
		}),
		megaport.WithUserAgent(userAgent),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Megaport API Client",
			"An unexpected error occurred when creating the Megaport API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Megaport Client Error: "+err.Error(),
		)
		return
	}

	_, err = megaportClient.Authorize(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Megaport API Client",
			"An unexpected error occurred when creating the Megaport API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Megaport Client Error: "+err.Error(),
		)
		return
	}

	// Parse AWS configuration
	var awsConfig awsConfig

	// First read environment variables for AWS configuration
	awsConfig.Region = os.Getenv("AWS_REGION")
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" && awsConfig.Region == "" {
		awsConfig.Region = region
	}

	awsConfig.AccessKey = os.Getenv("AWS_ACCESS_KEY_ID")
	awsConfig.SecretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	awsConfig.SessionToken = os.Getenv("AWS_SESSION_TOKEN")

	// Enable AWS integration if we have either credentials, profile, or assume role
	awsConfig.Enabled = (awsConfig.AccessKey != "" && awsConfig.SecretKey != "") ||
		os.Getenv("AWS_PROFILE") != "" ||
		os.Getenv("AWS_ROLE_ARN") != ""

	awsConfig.Profile = os.Getenv("AWS_PROFILE")
	awsConfig.AssumeRoleARN = os.Getenv("AWS_ROLE_ARN")
	awsConfig.ExternalID = os.Getenv("AWS_EXTERNAL_ID")

	if !config.AWSConfiguration.IsNull() && !config.AWSConfiguration.IsUnknown() {
		awsConfig.Enabled = true
		var awsConfigModel awsConfigurationModel

		diags = config.AWSConfiguration.As(ctx, &awsConfigModel, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Allow provider config to override environment variables
		if !awsConfigModel.AWSRegion.IsNull() {
			awsConfig.Region = awsConfigModel.AWSRegion.ValueString()
		}
		if !awsConfigModel.AWSAccessKey.IsNull() {
			awsConfig.AccessKey = awsConfigModel.AWSAccessKey.ValueString()
		}
		if !awsConfigModel.AWSSecretKey.IsNull() {
			awsConfig.SecretKey = awsConfigModel.AWSSecretKey.ValueString()
		}
		if !awsConfigModel.AWSSessionToken.IsNull() {
			awsConfig.SessionToken = awsConfigModel.AWSSessionToken.ValueString()
		}
		if !awsConfigModel.AWSProfile.IsNull() {
			awsConfig.Profile = awsConfigModel.AWSProfile.ValueString()
		}
		if !awsConfigModel.AWSAssumeRoleARN.IsNull() {
			awsConfig.AssumeRoleARN = awsConfigModel.AWSAssumeRoleARN.ValueString()
		}
		if !awsConfigModel.AWSExternalID.IsNull() {
			awsConfig.ExternalID = awsConfigModel.AWSExternalID.ValueString()
		}
	}

	// Add debug logging for AWS configuration
	ctx = tflog.SetField(ctx, "aws_enabled", awsConfig.Enabled)
	ctx = tflog.SetField(ctx, "aws_region", awsConfig.Region)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "aws_secret_key", "aws_session_token")
	tflog.Debug(ctx, "AWS integration configuration")

	// Add this after the AWS config parsing code
	// Validate AWS configuration when enabled
	if awsConfig.Enabled {
		if awsConfig.Region == "" {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("aws_region"),
				"Missing AWS region",
				"AWS integration is enabled but no region is specified. "+
					"Set the aws_region value in the configuration or use the AWS_REGION environment variable.",
			)
		}

		// Check if we have credentials (access key + secret key) OR profile OR assume role
		hasCredentials := awsConfig.AccessKey != "" && awsConfig.SecretKey != ""
		hasProfile := awsConfig.Profile != ""
		hasRole := awsConfig.AssumeRoleARN != ""

		if !hasCredentials && !hasProfile && !hasRole {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("aws_enabled"),
				"Missing AWS authentication",
				"AWS integration is enabled but no authentication method is specified. "+
					"Either provide access_key/secret_key, aws_profile, or aws_assume_role_arn.",
			)
		}

		tflog.Info(ctx, "AWS integration enabled", map[string]any{
			"region": awsConfig.Region,
			"auth_method": map[string]any{
				"static_credentials": hasCredentials,
				"profile":            hasProfile,
				"assume_role":        hasRole,
			},
		})
	}

	// Return provider data including AWS config
	providerData := &megaportProviderData{
		client:            megaportClient,
		awsConfig:         &awsConfig,
		cancelAtEndOfTerm: cancelAtEndOfTerm,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData

	tflog.Info(ctx, "Configured Megaport API client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *megaportProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewlocationDataSource,
		NewPartnerPortDataSource,
		NewMVEImageDataSource,
		NewMVESizeDataSource,
		NewMCRPrefixFilterListDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *megaportProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMCRResource,
		NewMCRPrefixFilterListResource,
		NewPortResource,
		NewLagPortResource,
		NewMVEResource,
		NewVXCResource,
		NewIXResource,
	}
}

func toResourceTagMap(ctx context.Context, in types.Map) (map[string]string, diag.Diagnostics) {
	tags := map[string]string{}
	diags := in.ElementsAs(ctx, &tags, false)
	return tags, diags
}

// Create AWS session based on configuration
func createAWSSession(config awsConfig) (*session.Session, error) {
	opts := session.Options{
		Config: aws.Config{
			Region: aws.String(config.Region),
		},
	}

	// If access key and secret key provided, use static credentials
	if config.AccessKey != "" && config.SecretKey != "" {
		opts.Config.Credentials = credentials.NewStaticCredentials(
			config.AccessKey,
			config.SecretKey,
			config.SessionToken,
		)
	} else if config.Profile != "" {
		// Use shared config
		opts.Profile = config.Profile
		opts.SharedConfigState = session.SharedConfigEnable
	}

	sess, err := session.NewSessionWithOptions(opts)
	if err != nil {
		return nil, err
	}

	// If role ARN is specified, assume the role
	if config.AssumeRoleARN != "" {
		// Import the AWS STS service for assuming roles
		stsClient := sts.New(sess)

		// Set up assume role input
		assumeRoleInput := &sts.AssumeRoleInput{
			RoleArn:         aws.String(config.AssumeRoleARN),
			RoleSessionName: aws.String("MegaportTerraformProvider"),
		}

		// Add external ID if provided
		if config.ExternalID != "" {
			assumeRoleInput.ExternalId = aws.String(config.ExternalID)
		}

		// Assume the role
		assumeRoleOutput, err := stsClient.AssumeRole(assumeRoleInput)
		if err != nil {
			return nil, fmt.Errorf("failed to assume role %s: %v", config.AssumeRoleARN, err)
		}

		// Create a new session with the temporary credentials from the assumed role
		newSessionConfig := &aws.Config{
			Region: aws.String(config.Region),
			Credentials: credentials.NewStaticCredentials(
				*assumeRoleOutput.Credentials.AccessKeyId,
				*assumeRoleOutput.Credentials.SecretAccessKey,
				*assumeRoleOutput.Credentials.SessionToken,
			),
		}

		return session.NewSession(newSessionConfig)
	}

	return sess, nil
}

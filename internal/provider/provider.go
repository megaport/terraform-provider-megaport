package provider

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	megaport "github.com/megaport/megaportgo"
)

// Set by Provider Config, is 10 minutes by default.
var waitForTime time.Duration

// megaportProviderModel maps provider schema data to a Go type.
type megaportProviderModel struct {
	Environment   types.String `tfsdk:"environment"`
	AccessKey     types.String `tfsdk:"access_key"`
	SecretKey     types.String `tfsdk:"secret_key"`
	TermsAccepted types.Bool   `tfsdk:"accept_purchase_terms"`
	WaitTime      types.Int64  `tfsdk:"wait_time"`
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &megaportProvider{}
)

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
					stringvalidator.OneOf("production", "staging"),
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
			"wait_time": schema.Int64Attribute{
				Description: "The time to wait in minutes for creating and updating resources in Megaport API. Default value is 10.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
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
	if environment == "" || environment == "staging" {
		megaportGoEnv = megaport.EnvironmentStaging
	} else if environment == "production" {
		megaportGoEnv = megaport.EnvironmentProduction
	} else {
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

	// Make the Megaport client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = megaportClient
	resp.ResourceData = megaportClient

	tflog.Info(ctx, "Configured Megaport API client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *megaportProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewlocationDataSource,
		NewPartnerPortDataSource,
		NewMVEImageDataSource,
		NewMVESizeDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *megaportProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMCRResource,
		NewPortResource,
		NewLagPortResource,
		NewMVEResource,
		NewVXCResource,
	}
}

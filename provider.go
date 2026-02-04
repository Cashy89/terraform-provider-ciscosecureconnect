package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("MERAKI_API_KEY", nil),
				Description: "API key for Meraki dashboard (can also be set with MERAKI_API_KEY).",
			},
			"base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://api.meraki.com/api/v1",
				Description: "Base URL for Meraki API.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"hbsecureconnect_secure_connect_site": resourceSecureConnectSite(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"hbsecureconnect_secure_connect_site": dataSourceSecureConnectSite(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	apiKey := d.Get("api_key").(string)
	baseURL := d.Get("base_url").(string)

	client := NewClient(apiKey, baseURL)
	return client, diags
}

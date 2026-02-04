package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSecureConnectSite() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSecureConnectSiteRead,
		Schema: map[string]*schema.Schema{
			"organization_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"site_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"region": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceSecureConnectSiteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Add context to logs
	tflog.Info(ctx, "Reading Secure Connect site data source")

	client := meta.(*MerakiClient)
	orgID := d.Get("organization_id").(string)
	siteName := d.Get("site_name").(string)

	// Pass context to the API call
	sites, err := client.GetSecureConnectSites(ctx, orgID)
	if err != nil {
		return diag.Errorf("error fetching sites: %v", err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Received %d sites from API", len(sites)), map[string]interface{}{
		"organization_id": orgID,
		"site_count":     len(sites),
	})

	var foundSite map[string]interface{}
	for _, site := range sites {
		name, ok := site["name"].(string)
		if !ok {
			tflog.Warn(ctx, "Site missing name field", map[string]interface{}{
				"site_id": site["id"],
			})
			continue
		}

		if name == siteName {
			if foundSite != nil {
				return diag.Errorf("multiple sites found with name %q", siteName)
			}
			foundSite = site
		}
	}

	if foundSite == nil {
		return diag.Errorf("no site found with name %q in organization %q", siteName, orgID)
	}

	// Set Terraform attributes
	if err := d.Set("id", foundSite["id"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region", foundSite["region"]); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(foundSite["id"].(string))

	tflog.Info(ctx, "Successfully found Secure Connect site", map[string]interface{}{
		"site_id": foundSite["id"],
		"name":    foundSite["name"],
	})

	return nil
}
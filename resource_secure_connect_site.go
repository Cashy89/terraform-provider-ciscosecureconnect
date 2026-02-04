package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceSecureConnectSite() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSecureConnectSiteCreate,
		ReadContext:   resourceSecureConnectSiteRead,
		UpdateContext: resourceSecureConnectSiteUpdate,
		DeleteContext: resourceSecureConnectSiteDelete,

		Schema: map[string]*schema.Schema{
			"organization_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Meraki Organization ID.",
			},
			"site_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Meraki Site (Network) ID.",
			},
			"region_type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"CNHE", "CloudHub"}, false),
				Description:  "Region type, either 'CNHE' or 'CloudHub'.",
			},
			"region_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional region ID to use.",
			},
			"region_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional name for a new region.",
			},
		},
	}
}

func resourceSecureConnectSiteCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*MerakiClient)

	orgID := d.Get("organization_id").(string)
	siteID := d.Get("site_id").(string)
	regionType := d.Get("region_type").(string)
	regionID := d.Get("region_id").(string)
	regionName := d.Get("region_name").(string)

	err := client.CreateSecureConnectSite(ctx, orgID, siteID, regionType, regionID, regionName)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s:%s", orgID, siteID))
	return resourceSecureConnectSiteRead(ctx, d, m)
}

func resourceSecureConnectSiteRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// API does not support direct read; you may optionally call a list and filter.
	return nil
}

func resourceSecureConnectSiteUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Not implemented – Meraki API doesn't support updates to enrollments.
	return nil
}

func resourceSecureConnectSiteDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*MerakiClient)

	orgID := d.Get("organization_id").(string)
	siteID := d.Get("site_id").(string)

	err := client.DeleteSecureConnectSites(ctx, orgID, siteID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

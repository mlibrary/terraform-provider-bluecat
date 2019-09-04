package bluecat

import (
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/umich-vci/golang-bluecat"
)

func dataSourceIP4Network() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIP4NetworkRead,
		Schema: map[string]*schema.Schema{
			"container_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"IP4Block", "IP4Network", "DHCP4Range", ""}, false),
			},
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"properties": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"cidr": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"allow_duplicate_host": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"inherit_allow_duplicate_host": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"ping_before_assign": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"inherit_ping_before_assign": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"reference": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"gateway": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"inherit_default_domains": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"default_view": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"inherit_default_view": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"inherit_dns_restrictions": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"addresses_in_use": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"addresses_free": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceIP4NetworkRead(d *schema.ResourceData, meta interface{}) error {
	mutex.Lock()
	client, err := meta.(*Config).Client()
	if err != nil {
		mutex.Unlock()
		return err
	}

	containerID, err := strconv.ParseInt(d.Get("container_id").(string), 10, 64)
	if err = bam.LogoutClientIfError(client, err, "Unable to convert container_id from string to int64"); err != nil {
		mutex.Unlock()
		return err
	}
	otype := d.Get("type").(string)
	address := d.Get("address").(string)

	resp, err := client.GetIPRangedByIP(containerID, otype, address)
	if err = bam.LogoutClientIfError(client, err, "Failed to get IP4 Networks by hint"); err != nil {
		mutex.Unlock()
		return err
	}

	id := *resp.Id
	name := *resp.Name
	otype = *resp.Type
	properties := *resp.Properties
	d.SetId(strconv.FormatInt(id, 10))
	d.Set("name", name)
	d.Set("properties", properties)
	d.Set("type", otype)

	props := strings.Split(properties, "|")
	for x := range props {
		if len(props[x]) > 0 {
			prop := strings.Split(props[x], "=")[0]
			val := strings.Split(props[x], "=")[1]

			switch prop {
			case "CIDR":
				netmask, err := strconv.ParseFloat(strings.Split(val, "/")[1], 64)
				if err = bam.LogoutClientIfError(client, err, "Failed to get IP4 Network netmask"); err != nil {
					mutex.Unlock()
					return err
				}
				addressCount := int(math.Pow(2, (32 - netmask)))

				resp, err := client.GetEntities(*resp.Id, "IP4Address", 0, addressCount)
				if err = bam.LogoutClientIfError(client, err, "Failed to get child IP4 Addresses"); err != nil {
					mutex.Unlock()
					return err
				}

				addressesInUse := len(resp.Item)
				addressesFree := addressCount - addressesInUse

				d.Set("addresses_in_use", addressesInUse)
				d.Set("addresses_free", addressesFree)
				d.Set("cidr", val)
			case "allowDuplicateHost":
				d.Set("allow_duplicate_host", val)
			case "inheritAllowDuplicateHost":
				b, err := strconv.ParseBool(val)
				if err = bam.LogoutClientIfError(client, err, "Unable to parse inheritAllowDuplicateHost to bool"); err != nil {
					mutex.Unlock()
					return err
				}
				d.Set("inherit_allow_duplicate_host", b)
			case "pingBeforeAssign":
				d.Set("ping_before_assign", val)
			case "inheritPingBeforeAssign":
				b, err := strconv.ParseBool(val)
				if err = bam.LogoutClientIfError(client, err, "Unable to parse inheritPingBeforeAssign to bool"); err != nil {
					mutex.Unlock()
					return err
				}
				d.Set("inherit_ping_before_assign", b)
			case "reference":
				d.Set("reference", val)
			case "gateway":
				d.Set("gateway", val)
			case "inheritDefaultDomains":
				b, err := strconv.ParseBool(val)
				if err = bam.LogoutClientIfError(client, err, "Unable to parse inheritDefaultDomains to bool"); err != nil {
					mutex.Unlock()
					return err
				}
				d.Set("inherit_default_domains", b)
			case "defaultView":
				d.Set("default_view", val)
			case "inheritDefaultView":
				b, err := strconv.ParseBool(val)
				if err = bam.LogoutClientIfError(client, err, "Unable to parse inheritDefaultView to bool"); err != nil {
					mutex.Unlock()
					return err
				}
				d.Set("inherit_default_view", b)
			case "inheritDNSRestrictions":
				b, err := strconv.ParseBool(val)
				if err = bam.LogoutClientIfError(client, err, "Unable to parse inheritDNSRestrictions to bool"); err != nil {
					mutex.Unlock()
					return err
				}
				d.Set("inherit_dns_restrictions", b)
			default:
				log.Printf("[WARN] Unknown IP4 Address Property: %s", prop)
			}
		}
	}

	// logout client
	if err := client.Logout(); err != nil {
		mutex.Unlock()
		return err
	}
	log.Printf("[INFO] BlueCat Logout was successful")
	mutex.Unlock()

	return nil
}
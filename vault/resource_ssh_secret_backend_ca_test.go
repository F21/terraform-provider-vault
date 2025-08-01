// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vault

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/hashicorp/terraform-provider-vault/internal/provider"
	"github.com/hashicorp/terraform-provider-vault/testutil"
)

func TestAccSSHSecretBackendCA_basic(t *testing.T) {
	backend := "ssh-" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(context.Background(), t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		CheckDestroy:             testAccCheckSSHSecretBackendCADestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSSHSecretBackendCAConfigGenerated(backend),
				Check:  testAccSSHSecretBackendCACheck(backend),
			},
		},
	})
}

func TestAccSSHSecretBackendCA_provided(t *testing.T) {
	backend := "ssh-" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(context.Background(), t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		CheckDestroy:             testAccCheckSSHSecretBackendCADestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSSHSecretBackendCAConfigProvided(backend),
				Check:  testAccSSHSecretBackendCACheck(backend),
			},
		},
	})
}

func TestAccSSHSecretBackend_import(t *testing.T) {
	backend := "ssh-" + acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(context.Background(), t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccSSHSecretBackendCAConfigGenerated(backend),
				Check:  testAccSSHSecretBackendCACheck(backend),
			},
			{
				ResourceName: "vault_ssh_secret_backend_ca.test",
				ImportState:  true,
				// state cannot be verified since generate_signing_key and private_key are not returned by the API
				ImportStateVerify: false,
			},
		},
	})
}

// TestAccSSHSecretBackendCA_Upgrade_key_type uses ExternalProviders (vault) to
// generate a state file with a previous version of the provider and then
// verify that there are no planned changes after migrating to an updated
// schema to validate the sshSecretBackendCAUpgradeV0 state upgrader.
func TestAccSSHSecretBackendCA_Upgrade_key_type(t *testing.T) {
	backend := "ssh-" + acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"vault": {
						// 4.2.0 does not have the key_type field
						VersionConstraint: "4.2.0",
						Source:            "hashicorp/vault",
					},
				},
				Config: testAccSSHSecretBackendCAConfigGenerated(backend),
				Check:  testAccSSHSecretBackendCACheck(backend),
			},
			{
				ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(context.Background(), t),
				Config:                   testAccSSHSecretBackendCAConfigGenerated(backend),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccSSHSecretBackendCA_managedKeys(t *testing.T) {
	backend := "ssh-" + acctest.RandString(10)
	accessKey, secretKey := testutil.GetTestAWSCreds(t)
	managedKeyName := acctest.RandomWithPrefix("kms-key")

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(context.Background(), t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			SkipIfAPIVersionLT(t, testProvider.Meta(), provider.VaultVersion120)
		},
		CheckDestroy: testAccCheckSSHSecretBackendCADestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSSHSecretBackendCAConfigManagedKey(backend, accessKey, secretKey, managedKeyName),
				Check:  testAccSSHSecretBackendCACheck(backend),
			},
		},
	})
}

func testAccCheckSSHSecretBackendCADestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vault_ssh_secret_backend_ca" {
			continue
		}

		client, e := provider.GetClient(rs.Primary, testProvider.Meta())
		if e != nil {
			return e
		}

		backend := rs.Primary.ID
		secret, err := client.Logical().Read(backend + "/config/ca")
		if err != nil {
			return err
		}
		if secret != nil {
			return fmt.Errorf("CA information still exists for backend %q", rs.Primary.ID)
		}
	}
	return nil
}

func testAccSSHSecretBackendCAConfigGenerated(backend string) string {
	return fmt.Sprintf(`
resource "vault_mount" "test" {
  type = "ssh"
  path = "%s"
  description = "SSH Secret backend"
}

resource "vault_ssh_secret_backend_ca" "test" {
  backend              = vault_mount.test.path
  generate_signing_key = true
}`, backend)
}

func testAccSSHSecretBackendCAConfigProvided(backend string) string {
	return fmt.Sprintf(`
resource "vault_mount" "test" {
  type = "ssh"
  path = "%s"
  description = "SSH Secret backend"
}

resource "vault_ssh_secret_backend_ca" "test" {
  backend     = vault_mount.test.path
  private_key = <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAu/5/sDSqVMV6USjgPkGcHM9X3ENtgMU4AFrKAMCV85qbGhgR
w9zJruvIxT695/0kLH7UqeEfMxlY4XBuHkWRuU4Djd5cALelHJ8zmG+ZlaRcrQjV
RM0Pvn2D3BQsTSWIIWSzLmLuaYOGBMrrgBlGDrVLW88pksYiPTr4BdxqK/TzOwLK
YjwlT+XV3HQLxr6a7+SHk3//PWqQhhIZs+uaOSsg5xSBuUx6EGJIqSWBUiBhB6PJ
5ndGVDRZkiSmul6lp/4WcuvAkXiKqHCRCnNAcBAAhFUEnL0JqQ9g5QmIwEGc+L4t
g3v4Qi+IUwlk8LRkkrcEgAjxi04JO2XXBPzlGQIDAQABAoIBAA6Dw9ATAOOyq5MA
mO+1mRwQVjRHcHj0wTIl0Frmg61fToJhQV3h+iBrTAEOqxLyVIyq7jh/jS0g09/0
Ekx8CphIEbYuaOQVScY/9HfchfsryYwClpTNUF3gywF+/TynnS8W207FjKrQ4NQV
5sDpMqOIE91uzULr0VDw8J1jOz9RdEuFL1SkUwrAH8Ki1DePrU3Bag+tHel3g/0u
DLYsw//cIQ66vUxW0JIHh0IB8WQlYC/IuE+GmLcHbfFpyRFRfrHqy/F0aLACQWRt
bCePdD953b3x9sCvrftOhkD/ar0RPInWgjSJ3yycsa5eIQQrXzgA0QDN3A7pHeqP
czUZk9ECgYEA7SmGtNGfIYsPdZ0T5X87CtzkMi0sNNHkgxegh5BzCxs4eExrPhQz
mfH2OZhmJv3cuMQWkCnRoW8JA/fUtBxN/nJcw8PeCdH2OgQxu2LfdiOXE35E4Jhh
+4GUSTd8/Qgg1QuSRAAWRhDcRUrUljiYQqJVf7/xGkp1vCs7RKSe4ycCgYEAyu0v
cY0www6K93tZU7EoP9zUoW+tkDVrEDq8WkXMorci+p9TdRH3o/jGp/sIcHv90oA3
nhnLEhf1DLTWMmjp/+DC0pHk7cWiOATWLda/6rt3pmfSyWxAaN4yXLX2zd7IQ2t/
5OgspE5FYnJ2AJay7inaSJBkF125f6wgKVFaHb8CgYEAqIHZ4X4Th/zLVjDuYyDc
baJ3TSOFhl4f8/kEqW28IAcOP4Nkq24lH9uorFGZO1kiy/Efav0boo1HJZegfPyj
egf922a+y9FwFtbGEzN0PPear1IHVGFRNSdjmgYf+5Ub5uPa4BADw3LVXzKFC9tY
a/f1sdhKUfjX4IQDD4m8Dv8CgYBvfz0HLiWxtwbiDfM5yegslsB55yu9RayK4Urm
at2SNf/RJsOrWnDvtlwopgSwEWCYTXzBsLhkO6eYELB0SDLyNeO14RWhE2sbToUD
8K/IYLLQStGFfKYzOIsBZ7WwzgzJBoLiGjOVH7B99BgkIKk1tOdL4ZItSIEIxmFx
clKKbwKBgG5E3n8E+miHJ/5WqZ3tCpiEAwn1Mwj5XFZnJ9LUq7fmHwr3IP3PkyF4
FfAMjrELscc0jJu32AbLZiI6/mwztMu9cN16owccnN3BwBd5JCqdGv8Lxi93rsha
DYdgQ3utnZSfVvB0VpmW0YBysI/vLa0+6b58ubme1Ko93AdJsP0e
-----END RSA PRIVATE KEY-----
EOF
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7/n+wNKpUxXpRKOA+QZwcz1fcQ22AxTgAWsoAwJXzmpsaGBHD3Mmu68jFPr3n/SQsftSp4R8zGVjhcG4eRZG5TgON3lwAt6UcnzOYb5mVpFytCNVEzQ++fYPcFCxNJYghZLMuYu5pg4YEyuuAGUYOtUtbzymSxiI9OvgF3Gor9PM7AspiPCVP5dXcdAvGvprv5IeTf/89apCGEhmz65o5KyDnFIG5THoQYkipJYFSIGEHo8nmd0ZUNFmSJKa6XqWn/hZy68CReIqocJEKc0BwEACEVQScvQmpD2DlCYjAQZz4vi2De/hCL4hTCWTwtGSStwSACPGLTgk7ZdcE/OUZ test@terraform-vault-provider.local"
}`, backend)
}

func testAccSSHSecretBackendCACheck(backend string) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttrSet("vault_mount.test", "description"),
		resource.TestCheckResourceAttrSet("vault_ssh_secret_backend_ca.test", "public_key"),
		resource.TestCheckResourceAttr("vault_ssh_secret_backend_ca.test", "backend", backend),
	)
}

func testAccSSHSecretBackendCAConfigManagedKey(backend, accessKey, secretKey, keyName string) string {

	return fmt.Sprintf(`
resource "vault_managed_keys" "test" {
	aws {
		name       = "%s"
		access_key = "%s"
		secret_key = "%s"
		key_bits   = "2048"
		key_type   = "RSA"
		kms_key    = "alias/tfvp-ssh-managed-key"
        allow_generate_key = true
	}
}

resource "vault_mount" "test" {
  type = "ssh"
  path = "%s"
  description = "SSH Secret backend"
  allowed_managed_keys      = [tolist(vault_managed_keys.test.aws)[0].name]
}

resource "vault_ssh_secret_backend_ca" "test" {
  backend              = vault_mount.test.path
  managed_key_id       = tolist(vault_managed_keys.test.aws)[0].uuid
}`, keyName, accessKey, secretKey, backend)
}

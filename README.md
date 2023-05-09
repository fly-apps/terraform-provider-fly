![Fly.io logo](/imgs/fly.png)

# fly.io terraform provider

[![Tests](https://github.com/fly-apps/terraform-provider-fly/actions/workflows/test.yml/badge.svg)](https://github.com/fly-apps/terraform-provider-fly/actions/workflows/test.yml)

### Resources
- app (stable, but apps will be deprecated soon. Begin to favor machines.)
- cert (stable)
- ip (stable)
- volume (stable)
- machines (beta)
- postgres (todo)

### Data sources
- app (stable)
- cert (stable)
- ip (stable)
- volume (stable)


### Acceptance Test Setup
To run acceptance tests for this provider some scaffolding is required.

1. Create a new organization in your Fly.io account to isolate the resources
    * e.g. `fly orgs create fly-terraform-ci`
    * take note of the org slug
2. Export environment variables
    * `export FLY_TF_TEST_ORG=<your-org-slug-from-step-1>`
    * `export FLY_TF_TEST_APP="acctestapp-$(head /dev/urandom | LC_ALL=C tr -dc A-Za-z0-9 | head -c 10)"`
2. Run the commands below to persist your config values. You will need direnv installed.
    * `$(cd infra && echo "fly_ci_org = \"${FLY_TF_TEST_ORG}\"\nfly_ci_app = \"${FLY_TF_TEST_APP}\"" > terraform.tfvars) && echo "export ASDF_DEFAULT_TOOL_VERSIONS_FILENAME=\$(pwd)/.tool-versions\nexport FLY_TF_TEST_ORG=${FLY_TF_TEST_ORG}\nexport FLY_TF_TEST_APP=${FLY_TF_TEST_APP}" > .envrc`
    * You should receive a direnv warning because .`envrc` changed and has new unapproved content. Run `direnv allow`.
3. Got to the infra directory and run `terraform apply` to create the scaffolding.
4. You should now be able to run `make` in the repo root to run tests.


### TODO

1. Build abstraction around querying local tunnel

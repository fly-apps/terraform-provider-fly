# fly.io terraform provider

[![Tests](https://github.com/fly-apps/terraform-provider-fly/actions/workflows/test.yml/badge.svg)](https://github.com/fly-apps/terraform-provider-fly/actions/workflows/test.yml)

**This project is not currently maintained, and is not a recommended method of deployment to Fly.io.**

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
3. Run the commands below to persist your config values. You will need direnv installed.
    * `echo "export FLY_TF_TEST_ORG=${FLY_TF_TEST_ORG}" >> .make-overrides`
    * `echo "export FLY_TF_TEST_APP=${FLY_TF_TEST_APP}" >> .make-overrides`
    * (If using ASDF) `echo "export ASDF_DEFAULT_TOOL_VERSIONS_FILENAME=$(pwd)/.tool-versions"`
    * `$(cd infra && echo "fly_ci_org = \"${FLY_TF_TEST_ORG}\"\nfly_ci_app = \"${FLY_TF_TEST_APP}\"" > terraform.tfvars)`
4. Got to the infra directory and run `terraform apply` to create the scaffolding.
5. You should now be able to run `make` in the repo root to run tests.
6. (Optional) set FLY_TF_TEST_REGION in `.make-overrides` to a region closer to you

### Building with Docker
If you do not have a local Go environment, you can build in a container. The binary will be placed in the root of the repository.

If you are not building for linux, set `GOOS` and `GOARCH` environment variables appropriately.

* `docker-compose up` (default, linux build)
* `GOOS=darwin GOARCH=arm64 docker-compose up` (m1 mac)
* `docker-compose up --build` if the version of golang has changed since the last run.

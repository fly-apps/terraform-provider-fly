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


### TODO

1. Build abstraction around querying local tunnel


### Build w/ Docker

```
docker build --pull -t provider-fly:latest .
docker run --rm --entrypoint=cat provider-fly:latest /out/terraform-provider-fly > terraform-provider-fly
```

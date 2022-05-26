# fly.io terraform provider

<div style="text-align: center;">

[![https://github.com/sponsors/DAlperin](https://img.shields.io/badge/sponsor-30363D?style=for-the-badge&logo=GitHub-Sponsors&logoColor=#EA4AAA)](https://github.com/sponsors/DAlperin)

</div>

> ⚠️ _If you are a company or individual who finds this useful and would like to see it continued to be developed please consider supporting me via GH sponsors (or [hiring me](https://dov.dev) for all your contract development needs!). I am a student so sponsoring me helps me find more time to work on open source. Thank you!_

## Status
I've been working with the fly team to eventually turn this into an officially mantained provider from fly. For now this will be the canonical location for docs/releases. The official provider will live [here](https://github.com/superfly/terraform-provider-fly) once it's ready but don't worry about it for now, when it becomes time to switch I'll do my best to make it clear.

### Resources
- app (stable, but apps will be deprecated soon. Begin to favor machines.)
- cert (stable)
- ip (stable)
- volume (stable)
- machines (beta)
  - missing:
    - in place updates
    - machine states
    - block on machine start
    - native wireguard tunnel
- postgres (todo)

### Data sources
- app (stable)
- cert (stable)
- ip (stable)
- volume (stable)


### TODO

1. Build abstraction around querying local tunnel

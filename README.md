Configuration precedence is:
1. Env vars. 
2. Pulumi native (CLI/stack or provider args)

Integration tests located in the `tests` package use real resources and require `CHERRY_AUTH_TOKEN` and `CHERRY_TEAM_ID` to be set.

Project BGP has the somewhat unintuitive behavior of not getting an ASN, until there's a server with BGP enabled in that project, even if project-scope BGP enabled.
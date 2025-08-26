# pulltainer

Keeps Portainer images updated by automatically re-pulling images and redeploying all stacks via the API

## How does this work?

pulltailer uses the Portainer API to simulate the "Re-pull image and redeploy" option when updating a stack, it does this for every stack that doesn't have the `PULLTAINER_IGNORE` environment varible present. As far as I can tell this is the same as doing `docker compose pull` and `docker compose up` for every stack. Containers aren't recreated unless a change to the Compose file is made or a new image has actually been found

`compose.yaml`

```yaml
name: "pulltainer"

services:
  pulltainer:
    environment:
      # Optional: custom cron schedule
      # PULLTAINER_CRON: "0 4 * * *"
      PULLTAINER_URL: "https://portainer.example.com"
      PULLTAINER_API_KEY: "ptr_o+EcmI+HyjUJXqrYUoU9YZAKSCyOWWnHPrOZTmwBoOU="
      # Optional: use Portainer Business Edition update checker to skip stacks not marked as outdated
      # PULLTAINER_BE_API: "false"
    image: "ghcr.io/eiqnepm/pulltainer:v0.1.0"
    restart: "unless-stopped"
```

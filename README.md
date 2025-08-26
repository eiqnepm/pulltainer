# pulltainer
Keeps Portainer images updated by automatically re-pulling images and redeploying all stacks via the API

## How does this work?
pulltailer uses the Portainer API to simulate the "Re-pull image and redeploy" option when updating a stack, it does this for every stack that doesn't have the `PULLTAINER_IGNORE` environment varible present. As far as I can tell this is the same as doing `docker compose pull` and `docker compose up` for every stack. Containers aren't recreated unless a change to the Compose file is made or a new image has actually been found

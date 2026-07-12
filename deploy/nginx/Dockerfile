# The stack's single entrypoint. Config is baked in (COPY) rather than
# bind-mounted, so it works on Portainer/Git deploys and SELinux hosts
# where host bind-mounts get denied.
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/nginx.conf

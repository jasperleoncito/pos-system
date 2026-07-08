# TLS certificates

`docker-compose.prod.yml` mounts this directory into nginx at
`/etc/nginx/certs`. Place your certificates here before starting the
production stack:

- `fullchain.pem` — certificate chain
- `privkey.pem` — private key

For Let's Encrypt: copy (or symlink) from
`/etc/letsencrypt/live/<domain>/`. For a smoke test only:

```bash
openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
  -keyout privkey.pem -out fullchain.pem -subj "/CN=localhost"
```

`*.pem` files are gitignored.

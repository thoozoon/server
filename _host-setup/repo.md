## Add secrets

Generate a secret in .ssh/comp3007_actions

```bash
ssh-keygen -t ed25519 -C "github-actions@comp3007" -f ~/.ssh/comp3007_actions
```

Put .ssh/comp3007_actions.pub in the .ssh of the host user.

Enter the secret at github. In "Settings → Secrets and variables → Actions → New repository secret", enter the public key from comp3007_actions.pub.

DEPLOY_HOST = comp3007-f25.scs.carleton.ca

DEPLOY_USER = comp3007

DEPLOY_KEY = contents of comp3007_deploy (private key)

(Optional) DEPLOY_PORT = 22 if nonstandard, change below.

If your server requires sudo for systemctl restart and your comp3007 user isn’t passwordless, configure sudoers:

```bash
# Allow comp3007 to restart only this unit without a password
echo 'comp3007 ALL=(root) NOPASSWD:/usr/bin/systemctl restart comp3007' | sudo tee /etc/sudoers.d/comp3007-restart
sudo chmod 440 /etc/sudoers.d/comp3007-restart
```

The deploy script already uses sudo systemctl restart comp3007.

## Actions workflow

Add file .github/workflows/deploy.yml:

```yml
name: Deploy to comp3007 host

on:
  push:
    branches: [master]
  workflow_dispatch: {}

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout (shallow)
        uses: actions/checkout@v4
        with:
          fetch-depth: 1

      - name: Install SSH client
        run: sudo apt-get update && sudo apt-get install -y openssh-client

      - name: Start SSH agent and add key
        uses: webfactory/ssh-agent@v0.9.0
        with:
          ssh-private-key: ${{ secrets.DEPLOY_KEY }}

      - name: Strict known_hosts (adds host key)
        run: |
          mkdir -p ~/.ssh
          ssh-keyscan -T 10 ${{ secrets.DEPLOY_HOST }} >> ~/.ssh/known_hosts
          chmod 600 ~/.ssh/known_hosts

      - name: Run deploy script on server
        env:
          HOST: ${{ secrets.DEPLOY_HOST }}
          USER: ${{ secrets.DEPLOY_USER }}
        run: |
          ssh -o BatchMode=yes $USER@$HOST 'bash -lc "/opt/comp3007/deploy.sh"'
```

## Testing

Manually trigger the workflow (Actions → Deploy to comp3007 host → Run workflow) or push a commit to master.

Watch logs in GitHub Actions and also on the server:

```bash
journalctl -u comp3007 -e -f
```

# Ansible playbook for automated setup
Make sure you have done the following before running the script:
- built caddy using go build -C ./cmd/caddy -o ../../caddy
- built flipcam using go build .

## On local machine
```shell
ansible-playbook playbook.yaml \
    -i inventory.yaml \
    -l localhost
```

## On remote machine
```shell
ansible-playbook playbook.yaml \
    -i inventory.yaml \
    -e 'ansible_host=ip_here' \
    -e 'ansible_ssh_user=ssh_username' \
    -l remote
```

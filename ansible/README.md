# Ansible playbook for automated setup

## On local machine
```shell
ansible-playbook playbook.yaml --diff \
    -i inventory.yaml \
    --ask-become-pass \
    -l localhost
```

## On remote machine
```shell
ansible-playbook playbook.yaml --diff \
    -i inventory.yaml \
    --ask-become-pass \
    -e 'ansible_host=ip_here' \
    -e 'ansible_ssh_user=ssh_username' \
    -l remote
```

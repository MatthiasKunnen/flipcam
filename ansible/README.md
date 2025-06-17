# Ansible playbook for automated setup

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

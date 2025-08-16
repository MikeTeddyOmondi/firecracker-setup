# Using the `Dockerfile`

## SSH into the container

ssh -p 2222 root@localhost

## Use Docker inside the container

docker exec -it microvm-base docker run hello-world

## Check systemd services

docker exec microvm-base systemctl status

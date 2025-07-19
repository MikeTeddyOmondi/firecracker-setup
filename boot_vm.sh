export API_SOCKET="/tmp/app.socket"
sudo rm $API_SOCKET
sudo ./setup/bin/firecracker --api-sock "${API_SOCKET}" --config-file config.json




sudo rm $API_SOCKET
export API_SOCKET="/tmp/app.socket"
sudo ./setup/firecracker --api-sock "${API_SOCKET}" --config-file config.json




# To start prometheus + grafana

`cd metrics`

`docker compose up -d --build`

### Note: remember to allow ports for Prometheus to see host.docker.internal:xxxx from within container

`sudo ufw allow 11001`
`sudo ufw allow 11002`
`sudo ufw allow 11003`

# Install Node-exporter

You'll need to install node exporter for monitoring

1. Download Node Exporter
As first step, you need to download the Node Exporter binary which is available for Linux in the official Prometheus website here. In the website, you will find a table with the list of available builds. Of our interest in this case, is the node_exporter build for Linux AMD64:

Node Exporter Ubuntu Linux

In this case the latest available version is the 1.7.0. Copy the .tar.gz URL and download it somewhere in your server using wget or cURL:

`wget https://github.com/prometheus/node_exporter/releases/download/v1.7.0/node_exporter-1.7.0.linux-amd64.tar.gz`

2. Extract Node Exporter and move binary
After downloading the latest version of Node Exporter, proceed to extract the content of the downloaded tar using the following command:

`tar xvf node_exporter-1.7.0.linux-amd64.tar.gz`
The content of the zip will be extracted in the current directory, the extracted directory will contain 3 files:

LICENSE (license text file)
node_exporter (binary)
NOTICE (license text file)
You only need to move the binary file node_exporter to the /usr/local/bin directory of your system. Switch to the node_exporter directory:

`cd node_exporter-1.7.0.linux-amd64`
And then copy the binary file with the following command:

`sudo cp node_exporter /usr/local/bin`
Then you can remove the directory that we created after extracting the zip file content:

# Exit current directory
`cd ..`

# Remove the extracted directory
`rm -rf ./node_exporter-1.7.0.linux-amd64`
3. Create Node Exporter User
As a good practice, create an user in the system for Node Exporter:

`sudo useradd --no-create-home --shell /bin/false node_exporter`
And set the owner of the binary node_exporter to the recently created user:

`sudo chown node_exporter:node_exporter /usr/local/bin/node_exporter`
4. Create and start the Node Exporter service
The Node Exporter service should always start when the server boots so it will always be available to be scrapped for information. Create the node_exporter.service file with nano:

`sudo nano /etc/systemd/system/node_exporter.service`
And paste the following content in the file:

```
[Unit]
Description=Node Exporter
Wants=network-online.target
After=network-online.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter
Restart=always
RestartSec=3

[Install]https://github.com/prometheus/node_exporter/releases/download
WantedBy=multi-user.target
```

Close nano and save the changes to the file. Proceed to reload the daemon with:

`sudo systemctl daemon-reload`
And finally enable the node_exporter service with the following command:

`sudo systemctl enable node_exporter`
And then start the service:

`sudo systemctl start node_exporter`

`sudo ufw allow 9090`
`sudo ufw allow 9100`

now go to `http://localhost:9100/metrics`
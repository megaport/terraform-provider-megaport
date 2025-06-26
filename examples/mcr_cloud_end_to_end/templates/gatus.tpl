#! /bin/bash

# Set hostname
sudo hostnamectl set-hostname ${name}
sudo systemctl restart systemd-resolved.service

# Enable SSH password auth and add user
sudo grep -r PasswordAuthentication /etc/ssh -l | xargs -n 1 sudo sed -i 's/#\s*PasswordAuthentication\s.*$/PasswordAuthentication yes/; s/^PasswordAuthentication\s*no$/PasswordAuthentication yes/'
sudo adduser workload
sudo echo "workload:${password}" | sudo /usr/sbin/chpasswd
sudo sed -i'' -e 's+\%sudo.*+\%sudo  ALL=(ALL) NOPASSWD: ALL+g' /etc/sudoers
sudo usermod -aG sudo workload
sudo service sshd restart

# Set logging
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1

# Install packages
sudo DEBIAN_FRONTEND=noninteractive apt-get clean
sudo DEBIAN_FRONTEND=noninteractive apt-get update && sudo DEBIAN_FRONTEND=noninteractive apt-get upgrade -y
sudo DEBIAN_FRONTEND=noninteractive apt-get install docker.io -y
sudo DEBIAN_FRONTEND=noninteractive apt-get install nginx -y
sudo DEBIAN_FRONTEND=noninteractive apt-get install apache2-utils -y
sudo DEBIAN_FRONTEND=noninteractive apt-get install net-tools -y
sudo systemctl start docker
sudo systemctl enable docker

# Create Gatus config
sudo cat > config.yaml << EOL
ui:
  header: "${cloud} gatus dashboard"
  title: "${cloud}"
web:
  port: 80
endpoints:
EOL

if [ ${cloud} == "AWS" ]; then
 gatus_1="Azure"
 gatus_2="Google"
fi

if [ ${cloud} == "Azure" ]; then
 gatus_1="AWS"
 gatus_2="Google"
fi

if [ ${cloud} == "Google" ]; then
 gatus_1="AWS"
 gatus_2="Azure"
fi

ITER=1
for endpoint in $(echo ${inter}|tr "," "\n"); 
do
    group=$(eval echo \$gatus_$ITER)
    sudo cat >> config.yaml << EOL
    - name: Port 443
      url: "tcp://$endpoint:443"
      client:
        insecure: false
        ignore-redirect: false
        timeout: 2s
      interval: ${interval}s
      group: $group
      conditions:
      - "[CONNECTED] == true"
    - name: Port 3306
      url: "tcp://$endpoint:3306"
      client:
        insecure: false
        ignore-redirect: false
        timeout: 2s
      interval: ${interval}s
      group: $group
      conditions:
      - "[CONNECTED] == true"
    - name: Port 1433
      url: "tcp://$endpoint:1433"
      client:
        insecure: false
        ignore-redirect: false
        timeout: 2s
      interval: ${interval}s
      group: $group
      conditions:
      - "[CONNECTED] == true"
    - name: Port 5000
      url: "tcp://$endpoint:5000"
      client:
        insecure: false
        ignore-redirect: false
        timeout: 2s
      interval: ${interval}s
      group: $group
      conditions:
      - "[CONNECTED] == true"
    - name: Port 50100
      url: "tcp://$endpoint:50100"
      client:
        insecure: false
        ignore-redirect: false
        timeout: 2s
      interval: ${interval}s
      group: $group
      conditions:
      - "[CONNECTED] == true"
EOL
    ((ITER++))
done

# Configure nginx
sudo sed -i 's/80/81/g' /etc/nginx/sites-available/default

cat << EOF > /etc/nginx/conf.d/default.conf
server {
    listen 443;
    listen 514;
    listen 1521;
    listen 8443;
    listen 30000-30041;
    listen 5000;
    listen 50100;
    listen 1433;
    listen 3306;

    error_page    500 502 503 504  /50x.html;

    location      / {
        root      html;
    }
}
EOF

sudo service nginx restart

# Start Gatus container
sudo docker run -d --restart unless-stopped --name gatus -p 80:80 --mount type=bind,source=/config.yaml,target=/config/config.yaml twinproduction/gatus:v5.12.1

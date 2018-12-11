package utils

var MasterStartup = `
#!/bin/bash
set -e
set -x

(
ARCH=amd64
VERSION=__VERSION__
CONTROL_PLANE_VERSION=${VERSION}
SERVICE_CIDR=__SVC_CIDR__
POD_CIDR=__POD_CIDR__
KUBELET_VERSION=${VERSION}
CLUSTER_DNS_DOMAIN=cluster.local
PRIVATEIP=$(hostname -i)
PUBLICIP=__PUBLICIP__
TOKEN=__TOKEN__
PORT=6443
MACHINE=__MACHINE__

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
touch /etc/apt/sources.list.d/kubernetes.list
sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'
apt-get update -y
apt-get install -y \
  socat \
  ebtables \
  apt-transport-https \
  cloud-utils \
  prips

function install_configure_docker () {
    # prevent docker from auto-starting
    echo "exit 101" > /usr/sbin/policy-rc.d
    chmod +x /usr/sbin/policy-rc.d
    trap "rm /usr/sbin/policy-rc.d" RETURN
    apt-get install -y docker.io
    echo 'DOCKER_OPTS="--iptables=false --ip-masq=false"' > /etc/default/docker
    systemctl daemon-reload
    systemctl enable docker
    systemctl start docker
}
install_configure_docker

# kubeadm uses 10th IP as DNS server
CLUSTER_DNS_SERVER=$(prips ${SERVICE_CIDR} | head -n 11 | tail -n 1)
# Our Debian packages have versions like "1.8.0-00" or "1.8.0-01". Do a prefix
# search based on our SemVer to find the right (newest) package version.
function getversion() {
    name=$1
    prefix=$2
    version=$(apt-cache madison $name | awk '{ print $3 }' | grep ^$prefix | head -n1)
    if [[ -z "$version" ]]; then
        echo Can\'t find package $name with prefix $prefix
        exit 1
    fi
    echo $version
}
KUBELET=$(getversion kubelet ${KUBELET_VERSION}-)
KUBEADM=$(getversion kubeadm ${KUBELET_VERSION}-)
apt-get install -y \
    kubelet=${KUBELET} \
    kubeadm=${KUBEADM}
chmod a+rx /usr/bin/kubeadm

# function cleanMaster() {
#
# }

# Override network args to use kubenet instead of cni, override Kubelet DNS args and
# add cloud provider args.
cat > /etc/default/kubelet <<EOF
KUBELET_EXTRA_ARGS="--network-plugin=kubenet"
KUBELET_EXTRA_ARGS+=" --cluster-dns=${CLUSTER_DNS_SERVER} --cluster-domain=${CLUSTER_DNS_DOMAIN}"
EOF
systemctl daemon-reload
systemctl restart kubelet.service

# Set up kubeadm config file to pass parameters to kubeadm init.
cat > /etc/kubernetes/kubeadm_config.yaml <<EOF
apiVersion: kubeadm.k8s.io/v1alpha2
kind: MasterConfiguration
api:
  advertiseAddress: ${PUBLICIP}
  bindPort: ${PORT}
networking:
  serviceSubnet: ${SERVICE_CIDR}
kubernetesVersion: v${CONTROL_PLANE_VERSION}
apiServerCertSANs:
- ${PUBLICIP}
- ${PRIVATEIP}
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: ${TOKEN}
apiServerExtraArgs:
  cloud-provider: cce
controllerManagerExtraArgs:
  allocate-node-cidrs: "true"
  #cloud-provider: cce
  cluster-cidr: ${POD_CIDR}
  service-cluster-ip-range: ${SERVICE_CIDR}
EOF

modprobe br_netfilter
kubeadm init --config /etc/kubernetes/kubeadm_config.yaml
mkdir -p $HOME/.kube
cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
chown $(id -u):$(id -g) $HOME/.kube/config

for tries in $(seq 1 60); do
    kubectl --kubeconfig /etc/kubernetes/kubelet.conf annotate --overwrite node $(hostname) machine=${MACHINE} && break
    sleep 1
done
echo done.
) 2>&1 | tee /var/log/startup.log
`

var NodeStartup = `
#!/bin/bash
set -e
set -x
(
ARCH=amd64
VERSION=__VERSION__
KUBELET_VERSION=${VERSION}
SERVICE_CIDR=__SVC_CIDR__
POD_CIDR=__POD_CIDR__
CLUSTER_DNS_DOMAIN=cluster.local
PRIVATEIP=$(hostname -i)
PUBLICIP=__PUBLICIP__
TOKEN=__TOKEN__
PORT=6443
MACHINE=__MACHINE__
MASTER=__MASTER__

apt-get update
apt-get install -y apt-transport-https prips
apt-key adv --keyserver hkp://keyserver.ubuntu.com --recv-keys F76221572C52609D
cat <<EOF > /etc/apt/sources.list.d/k8s.list
deb [arch=amd64] https://apt.dockerproject.org/repo ubuntu-xenial main
EOF
apt-get update
function install_configure_docker () {
    # prevent docker from auto-starting
    echo "exit 101" > /usr/sbin/policy-rc.d
    chmod +x /usr/sbin/policy-rc.d
    trap "rm /usr/sbin/policy-rc.d" RETURN
    apt-get install -y docker.io
    echo 'DOCKER_OPTS="--iptables=false --ip-masq=false"' > /etc/default/docker
    systemctl daemon-reload
    systemctl enable docker
    systemctl start docker
}
install_configure_docker
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF > /etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF
apt-get update
mkdir -p /etc/kubernetes/
cat > /etc/kubernetes/cloud-config <<EOF
EOF
# Our Debian packages have versions like "1.8.0-00" or "1.8.0-01". Do a prefix
# search based on our SemVer to find the right (newest) package version.
function getversion() {
	name=$1
	prefix=$2
	version=$(apt-cache madison $name | awk '{ print $3 }' | grep ^$prefix | head -n1)
	if [[ -z "$version" ]]; then
		echo Can\'t find package $name with prefix $prefix
		exit 1
	fi
	echo $version
}
KUBELET=$(getversion kubelet ${KUBELET_VERSION}-)
KUBEADM=$(getversion kubeadm ${KUBELET_VERSION}-)
KUBECTL=$(getversion kubectl ${KUBELET_VERSION}-)
apt-get install -y kubelet=${KUBELET} kubeadm=${KUBEADM} kubectl=${KUBECTL}
# kubeadm uses 10th IP as DNS server
CLUSTER_DNS_SERVER=$(prips ${SERVICE_CIDR} | head -n 11 | tail -n 1)
# Override network args to use kubenet instead of cni, override Kubelet DNS args and
# add cloud provider args.
cat > /etc/default/kubelet <<EOF
KUBELET_EXTRA_ARGS="--network-plugin=kubenet"
KUBELET_EXTRA_ARGS+=" --cluster-dns=${CLUSTER_DNS_SERVER} --cluster-domain=${CLUSTER_DNS_DOMAIN}"
EOF
systemctl daemon-reload
systemctl restart kubelet.service
kubeadm join --token "${TOKEN}" "${MASTER}:${PORT}" --ignore-preflight-errors=all --discovery-token-unsafe-skip-ca-verification
for tries in $(seq 1 60); do
	kubectl --kubeconfig /etc/kubernetes/kubelet.conf annotate --overwrite node $(hostname) machine=${MACHINE} && break
	sleep 1
done
echo done.
) 2>&1 | tee /var/log/startup.log
`

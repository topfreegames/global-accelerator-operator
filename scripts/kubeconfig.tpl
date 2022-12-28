apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ${CABASE64}
    server: ${SERVER}
  name: ${CLUSTERNAME}
contexts:
- context:
    cluster: ${CLUSTERNAME}
    user: global-accelerator-operator
  name: ${CLUSTERNAME}
kind: Config
preferences: {}
users:
- name: global-accelerator-operator
  user:
    token: ${TOKEN}
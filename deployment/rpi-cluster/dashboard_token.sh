kubectl  get secret $(kubectl get sa/dashboard-admin-sa -o jsonpath="{.secrets[0].name}") -o go-template="{{.data.token | base64decode}}" > token.txt
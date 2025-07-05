.PHONY: all apply delete

all: apply

apply:
    kubectl create namespace ms-service
	kubectl apply -f mongo-pvc.yaml
	kubectl apply -f mongo-deployment.yaml
	kubectl port-forward svc/mongodb 27017:27017 -n ms-service

delete:
	kubectl delete -f mongo-pvc.yaml
	kubectl delete -f mongo-deployment.yaml
	kubectl delete namespace ms-service

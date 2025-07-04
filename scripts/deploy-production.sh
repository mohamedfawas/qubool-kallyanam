#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Load environment variables
if [ -f .env ]; then
    echo -e "${BLUE}📋 Loading environment variables from .env${NC}"
    export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)
else
    echo -e "${RED}❌ .env file not found! Please copy .env.example to .env and configure it.${NC}"
    exit 1
fi

# Validate required variables
required_vars=("PROJECT_ID" "CLUSTER_NAME" "REGION" "DOMAIN")
for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo -e "${RED}❌ $var not set in .env file${NC}"
        exit 1
    fi
done

echo -e "${BLUE}🚀 Deploying Qubool Kallyanam to Production${NC}"
echo -e "${YELLOW}📦 Project: $PROJECT_ID${NC}"
echo -e "${YELLOW}☸️  Cluster: $CLUSTER_NAME${NC}"
echo -e "${YELLOW}🌍 Region: $REGION${NC}"
echo -e "${YELLOW}🌐 Domain: $DOMAIN${NC}"
echo ""

# Get cluster credentials
echo -e "${BLUE}🔐 Getting cluster credentials...${NC}"
if ! gcloud container clusters get-credentials $CLUSTER_NAME --region=$REGION --project=$PROJECT_ID; then
    echo -e "${RED}❌ Failed to get cluster credentials. Make sure the cluster exists and you have access.${NC}"
    exit 1
fi

# Check if kubectl is working
echo -e "${BLUE}🔍 Checking cluster connectivity...${NC}"
if ! kubectl cluster-info &>/dev/null; then
    echo -e "${RED}❌ Cannot connect to Kubernetes cluster${NC}"
    exit 1
fi

# Apply shared Kubernetes resources first
echo -e "${BLUE}📋 Applying shared Kubernetes resources...${NC}"

echo -e "${YELLOW}1️⃣  Applying namespace...${NC}"
if kubectl apply -f deploy/k8s/namespace.yaml; then
    echo -e "${GREEN}✅ Namespace applied${NC}"
else
    echo -e "${RED}❌ Failed to apply namespace${NC}"
    exit 1
fi

echo -e "${YELLOW}2️⃣  Applying secrets...${NC}"
if envsubst < deploy/k8s/secrets.yaml | kubectl apply -f -; then
    echo -e "${GREEN}✅ Secrets applied${NC}"
else
    echo -e "${RED}❌ Failed to apply secrets${NC}"
    exit 1
fi

echo -e "${YELLOW}3️⃣  Applying configmaps...${NC}"
if envsubst < deploy/k8s/configmap.yaml | kubectl apply -f -; then
    echo -e "${GREEN}✅ ConfigMaps applied${NC}"
else
    echo -e "${RED}❌ Failed to apply configmaps${NC}"
    exit 1
fi

echo -e "${YELLOW}4️⃣  Applying ingress...${NC}"
if envsubst < deploy/k8s/ingress.yaml | kubectl apply -f -; then
    echo -e "${GREEN}✅ Ingress applied${NC}"
else
    echo -e "${RED}❌ Failed to apply ingress${NC}"
    exit 1
fi

# Deploy services in dependency order
# Auth first (others depend on it), then supporting services, finally gateway (routes to all)
services=("auth" "user" "chat" "payment" "admin" "gateway")
namespace="qubool-kallyanam-production"

echo ""
echo -e "${BLUE}🏗️  Deploying microservices...${NC}"

# Deploy each service individually
for i in "${!services[@]}"; do
    service="${services[$i]}"
    service_num=$((i+5)) # Continue numbering from shared resources
    
    echo -e "${YELLOW}${service_num}️⃣  Deploying $service service...${NC}"
    
    # Apply service first (networking)
    echo -e "${BLUE}   📡 Applying $service service...${NC}"
    if kubectl apply -f deploy/k8s/$service/service.yaml; then
        echo -e "${GREEN}   ✅ $service service applied${NC}"
    else
        echo -e "${RED}   ❌ Failed to apply $service service${NC}"
        exit 1
    fi
    
    # Apply deployment (workload)
    echo -e "${BLUE}   🚀 Applying $service deployment...${NC}"
    if envsubst < deploy/k8s/$service/deployment.yaml | kubectl apply -f -; then
        echo -e "${GREEN}   ✅ $service deployment applied${NC}"
    else
        echo -e "${RED}   ❌ Failed to apply $service deployment${NC}"
        exit 1
    fi
    
    # Apply HPA (scaling)
    echo -e "${BLUE}   📈 Applying $service auto-scaling...${NC}"
    if kubectl apply -f deploy/k8s/$service/hpa.yaml; then
        echo -e "${GREEN}   ✅ $service HPA applied${NC}"
    else
        echo -e "${RED}   ❌ Failed to apply $service HPA${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ $service service deployed successfully${NC}"
    echo ""
done

# Wait for deployments to be ready
echo -e "${BLUE}⏳ Waiting for all deployments to be ready...${NC}"

deployments=("qubool-kallyanam-auth" "qubool-kallyanam-user" "qubool-kallyanam-chat" "qubool-kallyanam-payment" "qubool-kallyanam-admin" "qubool-kallyanam-gateway")

for deployment in "${deployments[@]}"; do
    echo -e "${YELLOW}⏳ Waiting for $deployment...${NC}"
    if kubectl rollout status deployment/$deployment -n $namespace --timeout=300s; then
        echo -e "${GREEN}✅ $deployment is ready${NC}"
    else
        echo -e "${RED}❌ $deployment failed to become ready${NC}"
        echo -e "${YELLOW}📋 Check logs: kubectl logs -f deployment/$deployment -n $namespace${NC}"
        echo -e "${YELLOW}📋 Describe pod: kubectl describe pods -l app=$deployment -n $namespace${NC}"
        # Continue with other deployments instead of exiting
    fi
done

# Check final status
echo ""
echo -e "${BLUE}📊 Final Deployment Status:${NC}"
echo -e "${YELLOW}Pods:${NC}"
kubectl get pods -n $namespace
echo ""
echo -e "${YELLOW}Services:${NC}"
kubectl get services -n $namespace
echo ""
echo -e "${YELLOW}Ingress:${NC}"
kubectl get ingress -n $namespace
echo ""
echo -e "${YELLOW}HPAs:${NC}"
kubectl get hpa -n $namespace

# Get external IP
echo ""
echo -e "${BLUE}🔍 Getting external IP address...${NC}"
external_ip=$(kubectl get ingress qubool-kallyanam-ingress -n $namespace -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")

if [ "$external_ip" != "pending" ] && [ -n "$external_ip" ]; then
    echo -e "${GREEN}🌐 External IP: $external_ip${NC}"
    echo -e "${GREEN}🚀 Your application will be available at: https://$DOMAIN${NC}"
    echo -e "${YELLOW}📝 Make sure your domain points to this IP address${NC}"
else
    echo -e "${YELLOW}⏳ External IP is still being assigned. Check later with:${NC}"
    echo -e "${BLUE}   kubectl get ingress -n $namespace${NC}"
fi

# Show service endpoints for debugging
echo ""
echo -e "${BLUE}🔍 Service endpoints for debugging:${NC}"
for service in "${services[@]}"; do
    service_name="qubool-kallyanam-$service-service"
    endpoint=$(kubectl get service $service_name -n $namespace -o jsonpath='{.spec.clusterIP}:{.spec.ports[0].port}' 2>/dev/null || echo "not found")
    echo -e "${YELLOW}   $service: $endpoint${NC}"
done

echo ""
echo -e "${GREEN}🎉 Qubool Kallyanam deployment completed successfully!${NC}"
echo ""
echo -e "${BLUE}📋 Useful commands:${NC}"
echo -e "${YELLOW}   Check status: make status${NC}"
echo -e "${YELLOW}   View logs: make logs${NC}"
echo -e "${YELLOW}   Scale info: make scale-status${NC}"
echo -e "${YELLOW}   Debug specific service: kubectl logs -f deployment/qubool-kallyanam-[SERVICE] -n $namespace${NC}"
echo -e "${YELLOW}   Port forward for testing: kubectl port-forward svc/qubool-kallyanam-gateway-service 8080:8080 -n $namespace${NC}"
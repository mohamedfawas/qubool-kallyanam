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
if [ -z "$PROJECT_ID" ]; then
    echo -e "${RED}❌ PROJECT_ID not set in .env file${NC}"
    exit 1
fi

# Set variables
VERSION=${VERSION:-$(date +%Y%m%d-%H%M%S)}

echo -e "${BLUE}🔨 Building Qubool Kallyanam Docker Images${NC}"
echo -e "${YELLOW}📦 Project ID: $PROJECT_ID${NC}"
echo -e "${YELLOW}🏷️  Version: $VERSION${NC}"
echo ""

# Configure Docker for GCR
echo -e "${BLUE}🔐 Configuring Docker for Google Container Registry...${NC}"
gcloud auth configure-docker --quiet

# Define services and their corresponding Dockerfiles
declare -A services=(
    ["gateway"]="Dockerfile.gateway"
    ["auth"]="Dockerfile.auth"
    ["user"]="Dockerfile.user"
    ["chat"]="Dockerfile.chat"
    ["payment"]="Dockerfile.payment"
    ["admin"]="Dockerfile.admin"
)

# Build all images in parallel
echo -e "${BLUE}🚀 Starting parallel builds...${NC}"
pids=()

for service in "${!services[@]}"; do
    dockerfile=${services[$service]}
    image_name="qubool-kallyanam-${service}"
    
    echo -e "${YELLOW}📦 Building $image_name using deploy/docker/$dockerfile...${NC}"
    
    # Submit build to Cloud Build in background
    gcloud builds submit \
        --project=$PROJECT_ID \
        --tag gcr.io/$PROJECT_ID/$image_name:$VERSION \
        --tag gcr.io/$PROJECT_ID/$image_name:latest \
        --file deploy/docker/$dockerfile \
        --timeout=15m \
        . &
    
    pids+=($!)
done

# Wait for all builds to complete
echo -e "${BLUE}⏳ Waiting for all builds to complete...${NC}"
failed_builds=0

for i in "${!pids[@]}"; do
    service_name="${!services[@]}"
    service_name=$(echo "${!services[@]}" | cut -d' ' -f$((i+1)))
    
    if wait ${pids[$i]}; then
        echo -e "${GREEN}✅ $service_name build completed successfully${NC}"
    else
        echo -e "${RED}❌ $service_name build failed${NC}"
        failed_builds=$((failed_builds + 1))
    fi
done

# Check results
if [ $failed_builds -eq 0 ]; then
    echo ""
    echo -e "${GREEN}🎉 All Qubool Kallyanam images built and pushed successfully!${NC}"
    echo -e "${BLUE}🔍 View images: gcloud container images list --repository=gcr.io/$PROJECT_ID${NC}"
    echo -e "${BLUE}📋 Next step: make deploy${NC}"
else
    echo ""
    echo -e "${RED}❌ $failed_builds build(s) failed. Please check the logs above.${NC}"
    exit 1
fi
## Deployment Guide for Go Web Server

This document provides instructions for deploying the Go web server to Google Cloud Run.

### Prerequisites

1. Google Cloud SDK installed and configured
2. Docker installed
3. A Google Cloud Project with billing enabled
4. Cloud Run API enabled in your project

### Building the Docker Image

The Dockerfile is configured to work with the project structure where the `website` directory is at the same level as the `server` directory.

### Build Command

**IMPORTANT**: You must run the Docker build command from the **parent directory** (the `3007` directory), NOT from the `server` directory. This is because the Dockerfile needs access to both the `server/` and `website/` directories.

```bash
# Make sure you're in the parent directory
cd /Users/howe/Documents/teaching/3007

# Verify you can see both directories
ls -la
# You should see both 'server/' and 'website/' directories

# Build the Docker image
docker build -t go-webserver -f server/Dockerfile .
```

Note: The build context is set to the parent directory (`.`) so that the Dockerfile can access both `server/` (for the Go code) and `website/` (for the content) directories.

### Local Testing

Test the container locally:

```bash
docker run -p 8080:8080 go-webserver
```

Visit `http://localhost:8080` to test the application.

### Deploying to Google Cloud Run

#### 1. Tag and Push to Google Container Registry

```bash
# Replace PROJECT_ID with your Google Cloud project ID
export PROJECT_ID=your-project-id

# Tag the image for GCR
docker tag go-webserver gcr.io/$PROJECT_ID/go-webserver

# Configure Docker to use gcloud as a credential helper
gcloud auth configure-docker

# Push the image
docker push gcr.io/$PROJECT_ID/go-webserver
```

#### 2. Deploy to Cloud Run

```bash
gcloud run deploy go-webserver \
  --image gcr.io/$PROJECT_ID/go-webserver \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080 \
  --memory 512Mi \
  --cpu 1 \
  --max-instances 10 \
  --set-env-vars="SITE_PASSWORD=your-secure-password"
```

#### 3. Alternative: Deploy with Artifact Registry

For newer projects, use Artifact Registry instead of Container Registry:

```bash
# Create an Artifact Registry repository (one-time setup)
gcloud artifacts repositories create go-webserver-repo \
  --repository-format=docker \
  --location=us-central1

# Configure Docker for Artifact Registry
gcloud auth configure-docker us-central1-docker.pkg.dev

# Tag and push to Artifact Registry
docker tag go-webserver us-central1-docker.pkg.dev/$PROJECT_ID/go-webserver-repo/go-webserver
docker push us-central1-docker.pkg.dev/$PROJECT_ID/go-webserver-repo/go-webserver

# Deploy from Artifact Registry
gcloud run deploy go-webserver \
  --image us-central1-docker.pkg.dev/$PROJECT_ID/go-webserver-repo/go-webserver \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080 \
  --memory 512Mi \
  --cpu 1 \
  --max-instances 10 \
  --set-env-vars="SITE_PASSWORD=your-secure-password"
```

### Environment Variables

The application supports the following environment variables:

- `PORT`: Server port (default: 8080)
- `SITE_DIR`: Directory containing website files (default: ./website)
- `SITE_PASSWORD`: Password for site authentication (default: ahsahbeequen)

### Security Considerations

1. **Change the default password**: Set a secure password using the `SITE_PASSWORD` environment variable
2. **HTTPS**: Cloud Run automatically provides HTTPS endpoints
3. **Authentication**: The application includes basic password authentication

### Monitoring and Logs

View logs in the Google Cloud Console:

```bash
gcloud run services logs tail go-webserver --region us-central1
```

### Updating the Deployment

To update the service with a new version:

1. Build and push a new Docker image
2. Deploy with the same command, or use:

```bash
gcloud run services update go-webserver \
  --image gcr.io/$PROJECT_ID/go-webserver \
  --region us-central1
```

### Troubleshooting

#### Common Issues

1. **Build Context Error**: Make sure to run `docker build` from the parent directory (`3007/`), not from `server/`. The command should be `docker build -t go-webserver -f server/Dockerfile .` (note the `-f` flag and final `.`)
2. **Website Files Missing**: Ensure the `website` directory exists in the build context
3. **Port Issues**: Cloud Run expects the application to listen on the port specified by the `PORT` environment variable
4. **Template Errors**: Verify that the `templates/` directory is included in the Docker image

#### Debug Commands

Check if the service is running:

```bash
gcloud run services list --region us-central1
```

Get service details:

```bash
gcloud run services describe go-webserver --region us-central1
```

View recent logs:

```bash
gcloud run services logs read go-webserver --region us-central1 --limit 50
```

### Testing the Build Locally

Before deploying to Cloud Run, test the Docker build process:

```bash
# Navigate to the correct directory
cd /Users/howe/Documents/teaching/3007

# Verify the directory structure
ls -la
# Should show both 'server/' and 'website/' directories

# Test the Docker build
docker build -t go-webserver-test -f server/Dockerfile .

# If successful, test locally
docker run -p 8080:8080 go-webserver-test
```

#### Build Troubleshooting

If you get file not found errors during build:

1. **Verify build context**: Make sure you're in the parent directory (`3007/`)
2. **Check file paths**: Ensure both `server/` and `website/` directories exist
3. **Verbose build**: Add `--progress=plain` to see detailed build output:

```bash
docker build --progress=plain -t go-webserver -f server/Dockerfile .
```

4. **Inspect build context**: See what files Docker is using:

```bash
docker build --no-cache -t go-webserver -f server/Dockerfile . 2>&1 | head -20
```

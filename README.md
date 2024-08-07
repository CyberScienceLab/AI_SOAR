# AI-SOAR

## Run AI_SOAR for first time
1) In terminal: cd Shuffler
2) Create .env file and copy contents from .env.example. Must assign values to following fields:
    -  GATEWAY_URL
3) Build backend image
    - In terminal: docker build -t csl-backend:latest /backend
4) Build frontend image
    - In terminal: docker build -t csl-frontend:latest /frontend
3) In terminal: docker compose up -d

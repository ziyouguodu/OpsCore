FROM node:24-alpine AS build

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
ARG VITE_API_BASE=/api
ARG VITE_ENABLE_DEMO_DATA=false
ENV VITE_API_BASE=$VITE_API_BASE
ENV VITE_ENABLE_DEMO_DATA=$VITE_ENABLE_DEMO_DATA
RUN npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80

FROM --platform=$BUILDPLATFORM node:24-alpine AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

ARG VITE_IM_API_BASE=/
ARG VITE_IM_WS_BASE=
ENV VITE_IM_API_BASE=$VITE_IM_API_BASE
ENV VITE_IM_WS_BASE=$VITE_IM_WS_BASE

RUN npm run build

FROM nginx:1.27-alpine AS runtime

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
      ffmpeg chromium fonts-dejavu-core fonts-liberation ca-certificates \
      libnss3 libatk1.0-0 libatk-bridge2.0-0 libcups2 libdrm2 libxkbcommon0 \
      libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libgbm1 libasound2 \
    && rm -rf /var/lib/apt/lists/*

COPY ttyd.x86_64 /usr/bin/ttyd
COPY vhs         /usr/bin/vhs
RUN chmod +x /usr/bin/ttyd /usr/bin/vhs

ENV ROD_BROWSER_BIN=/usr/bin/chromium
ENV VHS_NO_SANDBOX=true
WORKDIR /vhs
ENTRYPOINT ["/usr/bin/vhs"]

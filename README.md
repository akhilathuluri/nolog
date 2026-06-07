# 🔒 STEALTH ENGINE (Secure Chat)

Stealth Engine is a high-performance, **Zero-Knowledge**, End-to-End Encrypted (E2EE) terminal chat application built in Go. It operates exclusively over SSH, meaning no custom client installation is required—users simply connect using their native SSH terminal.

## ✨ Features

* **Zero-Client Architecture**: Connect using the native `ssh` command available on Mac, Linux, and Windows. No client binaries to download.
* **Military-Grade Cryptography**: 
  * **ChaCha20-Poly1305** Authenticated Encryption with Associated Data (AEAD).
  * **X25519** Elliptic-Curve Diffie-Hellman Key Exchange.
  * **Ed25519 Public-Key Signatures**: Cryptographic guarantees against Identity Spoofing.
  * **Split TX/RX Session Keys**: Independent transmit and receive keys for flawless bidirectional synchronization.
  * **Manual DH Rekey (Ctrl+K)**: True Post-Compromise Security. Instantly heal your session with a fresh Diffie-Hellman exchange.
* **Man-In-The-Middle (MITM) Protection**: Out-of-band QR code fingerprint verification for 1-to-1 connections.
* **Replay Attack Immunity**: Millisecond timestamp-based AAD sequence validation combined with a strictly tracked SHA-256 duplicate cache.
* **Group Chat Rooms**: Fully decentralized, cryptographically secure 1-to-N broadcasting using 256-bit symmetric Room Keys.
* **Secure File Sharing**: In-memory N-to-N file sharing over SCP using unpredictable, one-time cryptographic `Upload Tokens`. 
* **Server Resource Hardening**: Strict global memory quotas (250MB), 1000 session caps, and active file limits to prevent DoS/OOM attacks.
* **Encrypted Telemetry**: The server inherently blocks plaintext logging, encrypting all local telemetry into a binary file using a volatile, single-use ChaCha20-Poly1305 boot key.
* **Anti-Forensics**: All chat history and files live exclusively in volatile RAM. No databases, no logs, no persistence.
* **Flexible Authentication**: Native SSH password middleware to deter brute-force attacks, configurable via `.env`, flags, or environment variables.

---

## 🚀 Getting Started

### 1. Build the Server
Ensure you have Go 1.21+ installed on your system.
```bash
git clone https://github.com/yourusername/secure-chat.git
cd secure-chat
go mod tidy
go build -o secure-chat.exe
```

### 2. Start the Server
By default, the server binds to `localhost:23234`. 
If you want to protect your server against unauthorized connections, you can set a password. If a password is not set, the server operates openly and users will not be prompted for authentication.

You can set the password using three methods:

**Method 1: Command-Line Flag (Highest Priority)**
```bash
./secure-chat.exe -password "supersecret"
```

**Method 2: Environment Variable**
```bash
SECURE_CHAT_PASS="supersecret" ./secure-chat.exe
```

**Method 3: `.env` File**
Create a `.env` file in the same directory as the executable:
```env
SECURE_CHAT_PASS="supersecret"
```
Then simply run `./secure-chat.exe`.

### 3. Connect as a Client
Users connect to the server using their native SSH client. If you set a password, they will be prompted for it.
```bash
ssh -p 23234 localhost
```

---

## 📖 Usage Guide

When you connect to the server, you will be assigned an ephemeral 8-character Base58 `UniqueID`. 

### 1-to-1 Encrypted Chat
1. Ask your peer for their `UniqueID`.
2. Type their `UniqueID` into your terminal and press `Enter`.
3. The server will pair you, and your clients will execute an **X25519 Key Exchange**.
4. **Security Verification**: Both terminals will display a QR Code and a Fingerprint. Scan the QR code or read the fingerprint to your peer over an out-of-band channel (e.g., a phone call).
5. If the fingerprints match, press `Y` to establish the encrypted tunnel. If you press `N`, the application will instantly terminate to protect you from MITM attacks.

### Secure Group Rooms
1. Press `Ctrl+R` to generate a new Room. 
2. The client will generate a massive 32-byte cryptographic `RoomKey` and bundle it into a **Join Code**.
3. Press `Ctrl+Y` to copy the Join Code.
4. Send the Join Code to your friends securely. 
5. They press `Ctrl+J` and paste the Join Code.
6. *Note: Rooms automatically expire and are wiped from the server's memory exactly 10 minutes after creation.*

### Secure File Sharing (SCP Pipeline)
Because the TUI runs on the server, we use a custom **SCP-to-TUI pipeline** to allow users to securely upload local files directly into the encrypted chat stream without ever touching the server's hard drive.

**To Send a File:**
1. Check the right sidebar of your TUI for your personalized `Upload File` command.
2. Open a *new, local terminal* on your computer.
3. Run the SCP upload command, replacing `<file>` with your actual file path:
   ```bash
   scp -O -P 23234 my_picture.jpg localhost:upload_<YourUniqueID>
   ```
4. The server instantly intercepts the upload into RAM, encrypts it with ChaCha20-Poly1305, and broadcasts it into the chat room.

**To Receive a File:**
1. When someone uploads a file, a download notification will appear in your chat.
2. A personalized `scp download` command will appear on your right sidebar.
3. Open a *new, local terminal* on your computer and run the command to securely download the decrypted file directly to your hard drive.
4. *Note: Uploaded files are automatically purged from the server's memory after 10 minutes.*

### Reading Encrypted Server Logs
For maximum stealth, all server connection telemetry is stripped from standard output and encrypted into a binary `stealth.log` file using a volatile `ChaCha20-Poly1305` key printed uniquely upon each server boot. 

To decrypt and view these logs as an administrator, pass the printed 64-character hex key via the `--read-logs` flag:
```bash
./secure-chat.exe --read-logs <Your_64_Character_Hex_Key>
```

---

## 🌍 Production Deployment

Stealth Engine is a lightweight Go binary, meaning it can be deployed almost anywhere with minimal overhead.

### 1. Deploying on a VPS (Linux / Ubuntu)
The easiest way to host the server is to run it as a background service on a Virtual Private Server (VPS) like DigitalOcean, Linode, or AWS.

1. Build the Linux binary:
   ```bash
   env GOOS=linux GOARCH=amd64 go build -o secure-chat
   ```
2. Upload the binary to your VPS:
   ```bash
   scp secure-chat user@your-vps-ip:/usr/local/bin/
   ```
3. Create a `systemd` service file `/etc/systemd/system/secure-chat.service`:
   ```ini
   [Unit]
   Description=Stealth Engine Secure Chat
   After=network.target

   [Service]
   Type=simple
   User=root
   Environment="SECURE_CHAT_PASS=supersecret"
   ExecStart=/usr/local/bin/secure-chat
   Restart=always

   [Install]
   WantedBy=multi-user.target
   ```
4. Start and enable the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable secure-chat
   sudo systemctl start secure-chat
   ```

### 2. Deploying with Docker
You can easily containerize the application to run it in the cloud.

1. Create a simple `Dockerfile`:
   ```dockerfile
   FROM golang:1.21-alpine AS builder
   WORKDIR /app
   COPY . .
   RUN go build -o secure-chat

   FROM alpine:latest
   WORKDIR /app
   COPY --from=builder /app/secure-chat .
   EXPOSE 23234
   CMD ["./secure-chat"]
   ```
2. Build and run the image:
   ```bash
   docker build -t secure-chat .
   docker run -d -p 23234:23234 -e SECURE_CHAT_PASS="supersecret" secure-chat
   ```

### Port Forwarding
Ensure that port `23234` (or whatever port you bind to) is open on your server's firewall (e.g., `sudo ufw allow 23234/tcp`).

---

## 🛡️ Security Architecture

Stealth Engine is designed under the assumption that the server is compromised or malicious. 

1. **Zero-Knowledge Routing**: The server's `Hub` only routes `[]byte` arrays. It does not possess any private keys and cannot inspect or alter payloads.
2. **Key Wiping**: Shared secrets and private keys are aggressively zeroed from memory the millisecond they are no longer needed.
3. **Session Purging**: When a user disconnects or hits `Ctrl+Q` (Panic Exit), all their cipher engines, session pipes, and identities are instantly destroyed.
4. **Encrypted Telemetry**: All standard stdout logging is completely blocked. Administrative logs are scrambled into binary ciphertext via a volatile key generated on boot.
5. **Transport Layer Security**: The connection between the user's terminal and the server is encrypted via standard OpenSSH.

### Disclaimer
*This software is intended for educational purposes. While it implements industry-standard cryptography, it has not undergone a formal third-party security audit. Use at your own risk.*

# nolog

**Zero-knowledge, end-to-end encrypted terminal chat over SSH.**

nolog is a privacy-focused terminal chat platform that uses native SSH clients as the user interface. There are no custom desktop applications, browser clients, databases, or persistent chat histories. Users connect through standard SSH and establish cryptographically protected communication channels using modern end-to-end encryption.

### Highlights

* Zero-client architecture (native SSH only)
* End-to-end encrypted messaging
* X25519 key exchange
* Ed25519 identity signatures
* ChaCha20-Poly1305 authenticated encryption
* QR-code fingerprint verification
* Secure group rooms
* Secure file sharing
* RAM-only message routing
* No chat history persistence
* Optional password-protected access
* Cross-platform (Windows, Linux, macOS)

---

## Why nolog?

Most secure messaging platforms require dedicated applications, accounts, databases, and centralized infrastructure.

nolog takes a different approach:

```text
Native SSH Client
        │
        ▼
   SSH Server
        │
        ▼
 End-to-End Encryption
        │
        ▼
  RAM-Only Routing
```

Users simply connect using:

```bash
ssh -p 23234 your-server-ip
```

No client installation. No accounts. No message database.

---

## Features

### End-to-End Encryption

nolog establishes encrypted communication channels using:

* X25519 Elliptic Curve Diffie-Hellman key exchange
* Ed25519 digital signatures
* ChaCha20-Poly1305 authenticated encryption (AEAD)

The server acts only as a message router and does not possess users' private session keys.

### QR-Based Identity Verification

For direct conversations, participants verify each other's fingerprints using:

* QR codes
* Human-readable fingerprints

This helps detect man-in-the-middle attacks when verification is performed through an independent communication channel.

### Session Rekeying

Users can initiate a fresh Diffie-Hellman exchange at any time using:

```text
Ctrl + K
```

This generates new session keys without restarting the conversation.

### Secure Group Rooms

Create temporary encrypted rooms protected by randomly generated 256-bit room keys.

Features:

* Cryptographically generated room keys
* Shareable join codes
* Memory-only room storage
* Automatic expiration after 10 minutes

### Secure File Sharing

Files can be transferred through a custom SCP pipeline.

Example upload:

```bash
scp -O -P 23234 myfile.zip localhost:upload_<YourUniqueID>
```

Features:

* In-memory processing
* End-to-end encrypted transfers
* No persistent server-side storage
* Automatic cleanup

### Anti-Forensics Design

nolog intentionally avoids persistent storage:

* No databases
* No message history
* No persistent files
* No user accounts

All active communication data exists only in memory.

### Encrypted Telemetry

nolog minimizes operational logging and prevents plaintext telemetry storage.

Administrative telemetry is encrypted using a temporary ChaCha20-Poly1305 key generated at server startup.

A unique decryption key is displayed when the server boots. Store this key securely if you wish to inspect telemetry later.

#### Reading Encrypted Logs

Use the following command:

```bash
./nolog --read-logs <YOUR_64_CHARACTER_HEX_KEY>
```

Example:

```bash
./nolog --read-logs 6c2f9f2e7d4f0b6f8d4c7a2e9b1c3d4e5f6a7b8c9d0e1f23456789abcdef1234
```

The supplied key must match the telemetry encryption key generated during the server startup session.

#### Properties

- Telemetry is stored only in encrypted form.
- Plaintext logs are never written to disk.
- Each server boot generates a new encryption key.
- Previous telemetry cannot be decrypted without the corresponding key.

---

## Security Model

nolog is designed with the assumption that network infrastructure may be untrusted.

### Security Goals

nolog aims to provide:

* End-to-end encrypted messaging
* Confidential file transfers
* Protection against passive network monitoring
* Protection against server-side message inspection
* Identity verification through fingerprint matching
* Replay attack protections
* Ephemeral in-memory communication

### Replay Protection

nolog includes replay-attack protections through:

* Timestamp validation
* Duplicate message detection
* SHA-256 replay tracking

### Resource Hardening

Built-in safeguards include:

* Global memory limits
* Session limits
* File transfer limits
* Automatic cleanup mechanisms

These controls help reduce abuse and resource exhaustion risks.

---

## Threat Model

### Protects Against

✅ Passive network interception

✅ Server-side message inspection

✅ Message replay attempts

✅ Unauthorized room access without keys

✅ Identity impersonation (when fingerprint verification is performed)

### Does Not Protect Against

❌ Compromised user devices

❌ Keyloggers

❌ Screen capture malware

❌ Physical access attacks

❌ Traffic analysis and metadata observation

❌ Users who skip fingerprint verification

---

## Getting Started

### Requirements

* Go 1.21+
* OpenSSH client

### Clone Repository

```bash
git clone https://github.com/akhilathuluri/nolog.git
cd nolog
```

### Build

```bash
go mod tidy
go build -o nolog
```

### Run

Without password:

```bash
./nolog
```

With password:

```bash
./nolog -password "supersecret"
```

Environment variable:

```bash
SECURE_CHAT_PASS="supersecret" ./nolog
```

Or using a `.env` file:

```env
SECURE_CHAT_PASS="supersecret"
```

---

## Connecting

Connect using any SSH client:

```bash
ssh -p 23234 localhost
```

After connecting:

1. Receive a temporary UniqueID.
2. Exchange IDs with another user.
3. Establish a secure session.
4. Verify fingerprints.
5. Start chatting.

---

## Group Rooms

### Create Room

```text
Ctrl + R
```

### Copy Join Code

```text
Ctrl + Y
```

### Join Room

```text
Ctrl + J
```

Rooms automatically expire after 10 minutes.

---

## File Sharing

### Upload

```bash
scp -O -P 23234 myfile.jpg localhost:upload_<YourUniqueID>
```

### Download

Use the generated SCP command displayed in the terminal interface.

Uploaded files automatically expire after 10 minutes.

---

## Deployment

### Linux Server

Build:

```bash
GOOS=linux GOARCH=amd64 go build -o nolog
```

Upload:

```bash
scp nolog user@server:/usr/local/bin/
```

### systemd Service

```ini
[Unit]
Description=nolog Secure Chat
After=network.target

[Service]
Type=simple
User=root
Environment="SECURE_CHAT_PASS=supersecret"
ExecStart=/usr/local/bin/nolog
Restart=always

[Install]
WantedBy=multi-user.target
```

Enable:

```bash
sudo systemctl daemon-reload
sudo systemctl enable nolog
sudo systemctl start nolog
```

### Docker

Build:

```bash
docker build -t nolog .
```

Run:

```bash
docker run -d -p 23234:23234 -e SECURE_CHAT_PASS="supersecret" nolog
```

---

## Performance Characteristics

* Lightweight Go binary
* No database dependency
* Low memory footprint
* Concurrent connection handling
* Minimal deployment requirements

Suitable for:

* Self-hosted deployments
* Private communities
* Security research environments
* Internal team communication
* Temporary collaboration environments

---

## Security Status

nolog uses widely adopted cryptographic primitives and secure communication patterns. However:

* No independent third-party security audit has been completed.
* Cryptographic implementations should not be considered formally verified.
* Users should evaluate the software according to their own security requirements.

---

## Roadmap

* Automatic key rotation
* Multi-device identity support
* Optional Tor deployment guidance
* External security review
* Additional transport options
* Enhanced metadata resistance

---

## License

MIT License

---

## Disclaimer

This project is provided for educational and research purposes. While it implements modern cryptographic techniques, no guarantee of security is provided. Users should conduct their own review before deploying it in sensitive environments.

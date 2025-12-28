# Upgrading to v1.1.1 (One-Time Manual Update)

## For Users on v1.1.0 or Earlier

If you're currently using morpheus **v1.1.0 or earlier**, you need to manually update **one last time** to get the automatic update feature. After this, you'll never need to manually update again!

### Check Your Current Version

```bash
morpheus version
```

If you see `morpheus version 1.1.0` or earlier, follow the instructions below.

---

## 📱 Termux (Android) - Quick Update

**Option 1: Re-run the installer (Easiest)**

```bash
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

The installer will automatically update morpheus to the latest version.

**Option 2: Manual update**

```bash
cd ~/morpheus
git pull
make build
cp bin/morpheus ~/.local/bin/morpheus
```

---

## 💻 Desktop/Laptop - Manual Update

### If Installed to /usr/local/bin (most common)

```bash
cd /path/to/morpheus  # wherever you cloned it
git pull
make build
sudo make install
```

### If Using from bin/ directory

```bash
cd /path/to/morpheus
git pull
make build
# Now use ./bin/morpheus
```

---

## ✅ Verify Update

```bash
morpheus version
# Should show: morpheus version 1.1.1

morpheus check-update
# Should show: Already up to date: 1.1.1
```

---

## 🎉 You're Done!

From now on, you can update with just:

```bash
morpheus update
```

No more manual updates needed! The `update` command will:
- Check GitHub for new releases
- Show you release notes
- Ask for confirmation
- Automatically update morpheus

---

## Troubleshooting

### "Command not found: morpheus"

Your morpheus installation directory isn't in your PATH. Try:

```bash
# Find where morpheus is installed
which morpheus

# If not found, check common locations:
ls -la ~/.local/bin/morpheus
ls -la /usr/local/bin/morpheus
```

### "Permission denied" when updating

You need sudo for system-wide installations:

```bash
sudo make install
```

Or for Termux (no sudo needed):

```bash
make build
cp bin/morpheus ~/.local/bin/morpheus
```

### Git repository not found

You need to re-clone:

```bash
git clone https://github.com/nimsforest/morpheus.git
cd morpheus
make build
make install  # or: sudo make install
```

---

## Alternative: Fresh Install

If you run into issues, you can always do a fresh install:

**Desktop:**
```bash
rm -rf ~/morpheus  # or wherever you had it
git clone https://github.com/nimsforest/morpheus.git
cd morpheus
make build
sudo make install
```

**Termux:**
```bash
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

---

**After this one-time manual update to v1.1.1, you'll have the automatic update feature and never need to manually update again!** 🎉

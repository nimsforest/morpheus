# 📢 Announcement: Morpheus v1.1.1 with Automatic Updates

## For Users Currently on v1.1.0 or Earlier

Good news! Morpheus now has automatic updates. But first, you need to update **one last time manually**.

---

## 🔄 How to Update (One Last Time)

### Desktop/Laptop Users

```bash
cd /path/to/morpheus  # wherever you cloned it
git pull
make build
sudo make install
```

### Termux Users (Android)

**Easy way - just re-run the installer:**
```bash
curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash
```

**Or manually:**
```bash
cd ~/morpheus
git pull
make build
cp bin/morpheus ~/.local/bin/morpheus
```

---

## ✅ Verify You're Updated

```bash
morpheus version
# Should show: morpheus version 1.1.1
```

---

## 🎉 From Now On - It's Automatic!

After this one-time manual update, you can update with just:

```bash
morpheus update
```

That's it! No more:
- ❌ `git pull`
- ❌ `make build`
- ❌ `make install`
- ❌ Manual steps

Just one command: ✅ `morpheus update`

---

## 📖 More Information

- **Release notes:** https://github.com/nimsforest/morpheus/releases/tag/v1.1.1
- **Upgrade guide:** [UPGRADE_TO_v1.1.1.md](UPGRADE_TO_v1.1.1.md)
- **Documentation:** [README.md](README.md)

---

## ℹ️ What's New in v1.1.1

- `morpheus update` - Interactive update with release notes
- `morpheus check-update` - Check for updates without installing
- Automatic backup before updating
- Git-based updates from source
- Semantic version comparison

---

## 💬 Share This

**Short version for users:**

> Morpheus v1.1.1 is out with automatic updates! Update one last time manually:
> 
> Desktop: `cd morpheus && git pull && make build && sudo make install`  
> Termux: `curl -fsSL https://raw.githubusercontent.com/nimsforest/morpheus/main/scripts/install-termux.sh | bash`
> 
> Then verify: `morpheus version` (should show 1.1.1)
> 
> From now on, just run: `morpheus update` 🎉

---

**Questions?** Open an issue: https://github.com/nimsforest/morpheus/issues

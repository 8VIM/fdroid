# fdroid
This repository hosts an [F-Droid](https://f-droid.org/) repo for my apps. This allows you to install and update apps very easily.

### Apps

<!-- This table is auto-generated. Do not edit -->
| Icon | Name | Description | Version |
| --- | --- | --- | --- |
| <a href="https://github.com/8VIM/8VIM"><img src="fdroid/repo/inc.flide.vi8.pr509/en-US/icon.png" alt="8Vim Keyboard Debug PR: 509 icon" width="36px" height="36px"></a> | [**8Vim Keyboard Debug PR: 509**](https://github.com/8VIM/8VIM) | PR #509<br />Cursor movements was moving the cursor out of the input box<br /><br />refs: #426 | 0.17.0-pr.509-591911d478 |
| <a href="https://github.com/8VIM/8VIM"><img src="fdroid/repo/inc.flide.vi8.pr511/en-US/icon.png" alt="8Vim Keyboard Debug PR: 511 icon" width="36px" height="36px"></a> | [**8Vim Keyboard Debug PR: 511**](https://github.com/8VIM/8VIM) | PR #511<br />refs: #494 | 0.17.0-pr.511-4e64fe8e02 |
| <a href="https://github.com/8VIM/8VIM"><img src="fdroid/repo/inc.flide.vi8.rc/en-US/icon.png" alt="8Vim Keyboard RC icon" width="36px" height="36px"></a> | [**8Vim Keyboard RC**](https://github.com/8VIM/8VIM) | A Text Editor inside a keyboard, drawing it's inspiration from 8pen and Vim.  | 0.17.0-rc.20 |
| <a href="https://github.com/8VIM/8VIM"><img src="fdroid/repo/inc.flide.vi8.pr490/en-US/icon.png" alt="8vim_debug PR: 490 icon" width="36px" height="36px"></a> | [**8vim_debug PR: 490**](https://github.com/8VIM/8VIM) | PR #490<br />Add the ability for the keyboard to float around the screen | 0.17.0-pr.490-8bf8e93e18 |
<!-- end apps table --><!-- end apps table --><!-- end apps table -->

### How to use
1. At first, you should [install the F-Droid app](https://f-droid.org/), it's an alternative app store for Android.
2. Now you can copy the following [link](https://raw.githubusercontent.com/xarantolus/fdroid/main/fdroid/repo?fingerprint=080898ae4309aeceb58915e43a4b7c4a3e2cda40c91738e2c02f58339ab2fbd7), then add this repository to your F-Droid client:

    ```
    https://raw.githubusercontent.com/8VIM/fdroid/main/fdroid/repo?fingerprint=62c76ced86938b6f6a4c0e5f6992eb170a767487325171d74454ecd924c6408d
    ```

    Alternatively, you can also scan this QR code:

    <p align="center">
      <img src=".github/qrcode.png?raw=true" alt="F-Droid repo QR code"/>
    </p>

3. Open the link in F-Droid. It will ask you to add the repository. Everything should already be filled in correctly, so just press "OK".
4. You can now install my apps, e.g. start by searching for "Notality" in the F-Droid client.

Please note that some apps published here might contain [Anti-Features](https://f-droid.org/en/docs/Anti-Features/). If you can't find an app by searching for it, you can go to settings and enable "Include anti-feature apps".

### For developers
If you are a developer and want to publish your own apps right from GitHub Actions as an F-Droid repo, you can fork/copy this repo and see  [the documentation](setup.md) for more information on how to set it up.

### [License](LICENSE)
The license is for the files in this repository, *except* those in the `fdroid` directory. These files *might* be licensed differently; you can use an F-Droid client to get the details for each app.

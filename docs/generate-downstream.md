**Prerequisite:**
* rename ([install with Homebrew on OSX](https://formulae.brew.sh/formula/rename#default))

**Steps:**

1. Make the updates for the downstream code. We rename `koku` references to `costmanagement` in the downstream code.

    a. Checkout a branch based off main:
    ```
    $ git checkout -b <branch-name>
    ```

    b. Generate the code changes:
    ```
    $ make downstream
    ```

    c. Add/commit/push:
    ```
    $ git add/commit/push
    ```

3. Open PR against `downstream` to merge the downstream code changes.

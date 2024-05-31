**Prerequisite:**
* rename ([install with Homebrew on OSX](https://formulae.brew.sh/formula/rename#default))

**Steps:**

1. Create the `downstream-vX.Y.Z` branch based off main:
    ```
    git fetch origin
    git switch --no-track -c downstream-vX.Y.Z origin/main
    git push
    ```

2. Branch `downstream-vX.Y.Z` so we can make the updates for the downstream code. The only difference between upstream and downstream is the name of the API. We rename `koku` to `costmanagement` in the downstream code.

    a. Checkout a branch that will be merged into the `downstream-vX.Y.Z` branch:
    ```
    $ git checkout -b make-downstream-vX.Y.X (be sure to substitute the correct version for x.y.z, e.g. 2.0.0)
    ```

    b. Generate the code changes:
    ```
    $ make downstream
    ```

    c. Add/commit/push:
    ```
    $ git add/commit/push
    ```

3. Open PR against `downstream-vX.Y.Z` to merge the downstream code changes.

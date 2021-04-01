## Syncing the upstream with the downstream

To incorporate changes from the upstream branch (master) into the downstream branch (downstream), complete the following steps.

1. Create a branch and run the conversion command: 

    ```
    git checkout master
    git pull
    git checkout -b new-branch
    make downstream

    ```

2. Resolve any conflicts and submit a PR against the downstream branch. 
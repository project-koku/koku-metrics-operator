# Cost Management Metrics Operator (CMMO) Konflux Flow


## Helpful Links

* [Konflux internal release docs](https://konflux.pages.redhat.com/docs/users/releasing/preparing-for-release.html)
* [Cost Management Konflux docs](https://docs.google.com/document/d/1a8HWRrPxnW-CvaBqmzmmHeOwLuJgrVCKdz1v4OHkgaM/)
* [Konflux config folder for the Cost Management Metrics Operator](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/tree/main/tenants-config/cluster/stone-prd-rh01/tenants/cost-mgmt-dev-tenant/costmanagement-metrics-operator)


## Prerequisites:

- Application added in konflux


## Build:

#### Initial build
  
* Upon adding the component, a quay repo is created where the successful builds are pushed, see build repositories for [operator](https://quay.io/repository/redhat-user-workloads/cost-mgmt-dev-tenant/costmanagement-metrics-operator/costmanagement-metrics-operator?tab=tags&tag=latest) and [operator-bundle](https://quay.io/repository/redhat-user-workloads/cost-mgmt-dev-tenant/costmanagement-metrics-operator/costmanagement-metrics-operator-bundle?tab=tags&tag=latest). 

* Additionally, a pull request is opened by the red-hat-konflux bot on the repo which also triggers the first build. 

* See example pull request [#449](https://github.com/project-koku/koku-metrics-operator/pull/449).

#### Subsequent builds

* These are triggered based on the configuration in the `on-cel-expression` annotation in pipelinerun in the tekton folder for both push & pull requests. 

* For example, the pipelinerun for the operator bundle builds is triggered when the `bundle.dockerfile` and/or assets in the `bundle/` folder are modified.

#### Multi-architecture builds

* To ensure builds for all supported architectures, the [costmanagement-metrics-operator component](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/blob/main/tenants-config/cluster/stone-prd-rh01/tenants/cost-mgmt-dev-tenant/costmanagement-metrics-operator/operator.yaml?ref_type=heads#L9) was configured to use the [docker-build-multiplatform-oci-ta](https://github.com/konflux-ci/build-definitions/tree/main/pipelines/docker-build-multi-platform-oci-ta) pipeline.


## Test:

#### Integration Test Scenario (ITS)

* The CMMO [integration test](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/blob/main/tenants-config/cluster/stone-prd-rh01/tenants/cost-mgmt-dev-tenant/costmanagement-metrics-operator/integration-testing.yaml) is defined in the konflux-release-data repository.

* The default application testing context means an integration test pipeline run is triggered for every successful build per component in the application.

* A valid Enterprise Contract Policy need to be configured for the destination of the build and must pass before the pipeline can proceed. The CMMO is configured with [registry-standard](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/blob/main/config/common/product/EnterpriseContractPolicy/registry-standard.yaml) because it pushes content to registry.redhat.io.

* More on [testing application and components](https://konflux.pages.redhat.com/docs/users/how-tos/testing/index.html)


## Release:

### Pre-reqs

- A jira ticket for the CMMO version to be released, refer to previous [COST-5604](https://issues.redhat.com/browse/COST-5604).
- A snapshot that has passed the integration test pipeline, see example, [costmanagement-metrics-operator-dt9d7](https://console.redhat.com/application-pipeline/workspaces/cost-mgmt-dev/applications/costmanagement-metrics-operator/snapshots/costmanagement-metrics-operator-dt9d7/pipelineruns).


### Stage

#### 1. Configure release in konflux-release-data

In this section configure Konflux Red Hat instance to perform the release.

* Update `product_version` in [stage ReleasePlanAdmission](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/blob/main/config/stone-prd-rh01.pg1f.p1/product/ReleasePlanAdmission/cost-mgmt-dev/costmanagement-metrics-operator-staging.yaml).

* The CMMO is [released with an advisory](https://konflux.pages.redhat.com/docs/users/releasing/releasing-with-an-advisory.html) and additional relevant release notes such as issues and CVEs fixed should be listed in the `Release` object not in RPA.

Choose the appropriate advisory type:
    
* RHEA - Red Hat enhancement advisory. 
* RHBA - Red Hat bug advisory. Choose if fixing bug(s).
* RHSA - Red Hat security advisory. Choose if fixing CVE(s).

The advisories in this list are sorted from lowest to the highest priority. For instance always choose the RHSA if the release will fix a security tracker (a Jira ticket, see below).


#### 2. Release to stage

1. Release the main application containing operand, operator and bundle images. Make sure the bundle image contains up-to-date/latest pullspecs. The pullspecs should match the snapshot being released in Konflux. See example Release object in details.
    <details>   

      ```
      ---
      apiVersion: appstudio.redhat.com/v1alpha1
      kind: Release
      metadata:
        labels:
          release.appstudio.openshift.io/author: rh-ee-dnakabaa
        name: costmanagement-metrics-operator-3.3.2-staging-9
        namespace: cost-mgmt-dev-tenant
      spec:
        releasePlan: costmanagement-metrics-operator-staging
        snapshot: costmanagement-metrics-operator-dt9d7
        data:
          releaseNotes:
            cves:
            - component: costmanagement-metrics-operator
              key: CVE-2024-34155
              packages:
              - go/parser
            issues:
              fixed:
              - id: COST-5544
                source: issues.redhat.com
              - id: COST-5533
                source: issues.redhat.com
            references:
            - https://access.redhat.com/security/updates/classification
            - https://access.redhat.com/security/cve/CVE-2024-34155
            - https://docs.redhat.com/en/documentation/cost_management_service/1-latest/html/getting_started_with_cost_management/steps-to-cost-management
            topic: Cost Management Metrics Operator version 3.3.2 release.
            type: RHSA
      ```
    </details>
    

2. Check if the advisory was created in 
Konflux Advisories, [example advisory](https://gitlab.cee.redhat.com/rhtap-release/advisories/-/blob/main/data/advisories/cost-mgmt-dev-tenant/2024/10259/advisory.yaml).

3. Follow the [docs](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/README.md) in the Cost Management Metrics Operator FBC reposity to update catalog with released bundle.

4. Catalog applications are automatically released to stage on successful build and test pipeline runs.


#### 3. Test the release

Once the images are released in stage, follow the [instructions to gather the FBC releases](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/README.md#gather-fbc-for-qe) and pass them to QE for testing. Note that all releases must have `Succeeded`.


### Production

#### 1. Additional checks

* Release repositories must exist in [comet](https://comet.engineering.redhat.com/containers/repositories) for both the operator and operator bundle.
  
* If you move to a new universal base image, a new repository that matches that rhel version is required. That can be created via a merge request in [Pyxis Repo Configs](https://gitlab.cee.redhat.com/releng/pyxis-repo-configs/-/tree/main/products/costmanagement-metrics).

#### 2. Configure release in konflux-release-data

* Update `product_version` in [prod ReleasePlanAdmission](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/blob/main/config/stone-prd-rh01.pg1f.p1/product/ReleasePlanAdmission/cost-mgmt-dev/costmanagement-metrics-operator-prod.yaml)

* The CMMO is [released with an advisory](https://konflux.pages.redhat.com/docs/users/releasing/releasing-with-an-advisory.html) and additional relevant release notes such as issues and CVEs fixed should be listed in the `Release` object not in RPA.


#### 3. Release operator upon successful QE testing

- TBD


## Get Help

* For help with konflux related questions, reach out to [#konflux-users](https://redhat.enterprise.slack.com/archives/C04PZ7H0VA8) slack channel.

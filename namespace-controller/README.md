# Annotation Admission Controller for Kubernetes

The controller will automatically annotate namespace with correct project id, e.g:
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "development3",
    "labels": {
      "name": "development3"
    },
    "annotations": {
       "field.cattle.io/projectId": "c-jb5zn:p-8lwft"
    }
  }
}

Note the field.cattle.io/projectId annotation above.

A namespace create request will not be annotated if the user is member of multiple projects,
or if the user is a project owner of a project.

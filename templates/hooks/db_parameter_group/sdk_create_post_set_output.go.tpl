    // We need to instantly requeue after a create operation, otherwise the controller
    // will override the latest resource with the desired resource overrideParameters
    // and cause the controller to not properly compare latest and desired resources.
    return &resource{ko}, requeueWaitWhileCreating
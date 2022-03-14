    // If there was no error during update, requeue the request.
    // When ModifyDBInstance API is successful, it asynchronously
    // updates the DBInstanceStatus. Requeue to find the current
    // DBInstance status and set Synced condition accordingly
    if err == nil {
        return &resource{ko}, customWaitAtferUpdate
    }

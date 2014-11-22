if (req.request == "PURGE") {
    if (!client.ip ~ purge) {
        error 405 "Not allowed.";
    }

    return (lookup);
}

return (pass);

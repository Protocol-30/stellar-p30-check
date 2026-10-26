    case STELLAR_VALUE_SIGNED_MS:
        struct
        {
            TimePointMs closeTimeMs; // closeTime == closeTimeMs / 1000
            LedgerCloseValueSignature lcValueSignature;
        } signedMsValue;

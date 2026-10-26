    case STELLAR_VALUE_EMPTY_TX_SET_MS:
        struct
        {
            TimePointMs closeTimeMs; // closeTime == closeTimeMs / 1000
            Hash txSetHash;
            Hash previousLedgerHash;
            uint32 previousLedgerVersion;
            LedgerCloseValueSignature lcValueSignature;
        } proposedMsValue;

import React, { useEffect, useState } from 'react';
import axios from 'axios';

interface Field {
    name: string;
    value: string;
}

const FieldViewer: React.FC<{ fileName: string }> = ({ fileName }) => {
    const [fields, setFields] = useState<Field[]>([]);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        axios
            .get(`http://localhost:8080/fields/${fileName}`)
            .then(res => {
                setFields(res.data);
                setError(null);
            })
            .catch(err => {
                setFields([]);
                setError(err.response?.data || 'Could not extract fields');
            });
    }, [fileName]);

    return (
        <div>
            <h2>Form Fields</h2>
            {error ? (
                <p style={{ color: 'red' }}>⚠️ {error}</p>
            ) : fields.length === 0 ? (
                <p>PDF has no form fields.</p>
            ) : (
                <ul>
                    {fields.map(f => (
                        <li key={f.name}>
                            <strong>{f.name}</strong>: {f.value}
                        </li>
                    ))}
                </ul>
            )}
        </div>
    );
};

export default FieldViewer;

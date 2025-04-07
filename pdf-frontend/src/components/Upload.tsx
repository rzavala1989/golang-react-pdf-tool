import React from 'react';
import axios from 'axios';

interface Props {
    onUploaded: (fileName: string) => void;
}

const Upload: React.FC<Props> = ({ onUploaded }) => {
    const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
        if (!e.target.files?.length) return;

        const formData = new FormData();
        formData.append('pdf', e.target.files[0]);

        const res = await axios.post('http://localhost:8080/upload', formData);
        if (res.status === 200) {
            onUploaded(e.target.files[0].name);
        }
    };

    return (
        <div>
            <h2>Upload PDF</h2>
            <input type="file" accept="application/pdf" onChange={handleUpload} />
        </div>
    );
};

export default Upload;

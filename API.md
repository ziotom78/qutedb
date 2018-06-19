# RESTful API for QuTeDB

- `/api/v1/acquisitions` returns a list (in JSON format) containing metadata about all the acquisitions in the database
- `/api/v1/acquisitions/NN` returns details about the acquisition with ID NN (a number), in JSON format
- `/api/v1/acquisitions/NN/rawdata` returns a list (in JSON format) describing all the FITS file containing the raw data for the given acquisition
- `/api/v1/acquisitions/NN/rawdata/MM` returns the MM-th FITS file containing raw data for ASIC MM
- `/api/v1/acquisitions/NN/sumdata` returns a list (in JSON format) describing all the FITS file containing the scientific data for the given acquisition
- `/api/v1/acquisitions/NN/sumdata/MM` returns the MM-th FITS file containing scientific data for ASIC MM
- `/api/v1/acquisitions/NN/asichk` returns the FITS file containing ASIC housekeeping values
- `/api/v1/acquisitions/NN/externhk` returns the FITS file containing extern housekeeping values
- `/api/v1/acquisitions/NN/cryostathk` returns the FITS file containing cryostat housekeeping values

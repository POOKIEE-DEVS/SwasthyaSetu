/**
 * Get a document ready to upload: PDFs as they are, photos re-encoded as
 * JPEG and scaled down when they are large or in a format the server
 * doesn't accept (e.g. HEIC from iPhones). Phone camera photos are often
 * 5-10 MB; after this they are usually a few hundred KB, which keeps
 * uploads fast on mobile data and the free database small.
 */

export const MAX_UPLOAD_BYTES = 5 * 1024 * 1024;
const MAX_DIMENSION = 1800;
const PASS_THROUGH_BYTES = 1.5 * 1024 * 1024;
const ACCEPTED_IMAGES = ["image/jpeg", "image/png", "image/webp"];

export class UploadError extends Error {}

async function toJpeg(file: File): Promise<File> {
  let bitmap: ImageBitmap;
  try {
    bitmap = await createImageBitmap(file);
  } catch {
    throw new UploadError("This photo format isn't supported. Please choose a JPEG or PNG.");
  }
  const scale = Math.min(1, MAX_DIMENSION / Math.max(bitmap.width, bitmap.height));
  const canvas = document.createElement("canvas");
  canvas.width = Math.round(bitmap.width * scale);
  canvas.height = Math.round(bitmap.height * scale);
  canvas.getContext("2d")!.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
  bitmap.close();
  const blob = await new Promise<Blob | null>((resolve) =>
    canvas.toBlob(resolve, "image/jpeg", 0.85),
  );
  if (!blob) throw new UploadError("Couldn't process this photo. Please try another.");
  return new File([blob], file.name.replace(/\.[^.]+$/, "") + ".jpg", { type: "image/jpeg" });
}

export async function prepareUpload(file: File, { allowPdf = true } = {}): Promise<File> {
  if (file.type === "application/pdf") {
    if (!allowPdf) throw new UploadError("Please choose a photo, not a PDF.");
    if (file.size > MAX_UPLOAD_BYTES) throw new UploadError("This PDF is larger than 5 MB.");
    return file;
  }
  if (!file.type.startsWith("image/")) {
    throw new UploadError(allowPdf ? "Please choose a photo or a PDF." : "Please choose a photo.");
  }
  if (ACCEPTED_IMAGES.includes(file.type) && file.size <= PASS_THROUGH_BYTES) return file;
  const jpeg = await toJpeg(file);
  if (jpeg.size > MAX_UPLOAD_BYTES) throw new UploadError("This photo is still too large.");
  return jpeg;
}

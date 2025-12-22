export const download = (url: string, filename: string, type: string) => {
  const element = document.createElement("a");
  const file = new Blob([url], { type });
  element.href = URL.createObjectURL(file);
  element.download = filename;
  element.click();
};